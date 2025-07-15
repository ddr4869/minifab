package peer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
)

// ChannelConfig 채널 구성 정보
type ChannelConfig struct {
	ChannelID      string            `json:"channel_id"`
	Consortium     string            `json:"consortium"`
	OrdererAddress string            `json:"orderer_address"`
	Organizations  []string          `json:"organizations"`
	Capabilities   map[string]bool   `json:"capabilities"`
	Policies       map[string]string `json:"policies"`
	Version        uint64            `json:"version"`
}

// ChannelGenesisBlock 채널 생성 블록
type ChannelGenesisBlock struct {
	Number       uint64         `json:"number"`
	PreviousHash []byte         `json:"previous_hash"`
	Data         []byte         `json:"data"`
	Config       *ChannelConfig `json:"config"`
	Timestamp    time.Time      `json:"timestamp"`
}

// ChannelManager 채널 관리자
type ChannelManager struct {
	channels map[string]*Channel
	mutex    sync.RWMutex
}

// NewChannelManager 새로운 채널 관리자 생성
func NewChannelManager() *ChannelManager {
	cm := &ChannelManager{
		channels: make(map[string]*Channel),
	}

	// 저장된 채널 정보 로드
	cm.loadExistingChannels()

	return cm
}

func (cm *ChannelManager) loadExistingChannels() {
	channelsDir := "channels"

	// channels 디렉토리가 없으면 생성
	if _, err := os.Stat(channelsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(channelsDir, 0755); err != nil {
			logger.Warnf("Failed to create channels directory: %v", err)
			return
		}
		return
	}

	// 디렉토리 내의 파일들을 읽기
	files, err := os.ReadDir(channelsDir)
	if err != nil {
		logger.Warnf("Failed to load channels: %v", err)
		return
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			channelName := file.Name()[:len(file.Name())-5] // .json 확장자 제거

			// 파일에서 채널 데이터 읽기
			filePath := filepath.Join(channelsDir, file.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				logger.Warnf("Failed to read channel file %s: %v", file.Name(), err)
				continue
			}

			// JSON 언마샬
			var channel Channel
			if err := json.Unmarshal(data, &channel); err != nil {
				logger.Warnf("Failed to unmarshal channel data from %s: %v", file.Name(), err)
				continue
			}

			// MSP 복원
			if channel.MSPConfig != nil {
				fabricMSP := msp.NewFabricMSP()
				if err := fabricMSP.Setup(channel.MSPConfig); err != nil {
					logger.Warnf("Failed to setup MSP for channel %s: %v", channel.Name, err)
				} else {
					channel.MSP = fabricMSP
				}
			}

			cm.channels[channelName] = &channel
		}
	}
}

// saveChannel 채널 정보 저장
func (cm *ChannelManager) saveChannel(channel *Channel) error {
	// 채널 정보 저장 디렉토리 생성
	channelDir := "channels"
	if err := os.MkdirAll(channelDir, 0755); err != nil {
		return fmt.Errorf("failed to create channel directory: %v", err)
	}

	// 채널 정보 직렬화
	data, err := json.MarshalIndent(channel, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal channel data: %v", err)
	}

	// 채널 정보 파일 저장
	channelFile := filepath.Join(channelDir, channel.Name+".json")
	if err := os.WriteFile(channelFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write channel file: %v", err)
	}

	return nil
}

// CreateChannelConfig 채널 구성 정보 생성
func (cm *ChannelManager) CreateChannelConfig(channelID string, consortium string, ordererAddress string) (*ChannelConfig, error) {
	config := &ChannelConfig{
		ChannelID:      channelID,
		Consortium:     consortium,
		OrdererAddress: ordererAddress,
		Organizations:  make([]string, 0),
		Capabilities: map[string]bool{
			"V2_0": true,
		},
		Policies: map[string]string{
			"Readers": "ANY",
			"Writers": "ANY",
			"Admins":  "ANY",
		},
		Version: 1,
	}
	return config, nil
}

// CreateGenesisBlock 채널 생성 블록 생성
func (cm *ChannelManager) CreateGenesisBlock(config *ChannelConfig) (*ChannelGenesisBlock, error) {
	// 채널 구성 정보를 JSON으로 직렬화
	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal channel config: %v", err)
	}

	// 이전 해시 계산 (제네시스 블록이므로 빈 바이트 배열)
	previousHash := sha256.Sum256([]byte{})

	block := &ChannelGenesisBlock{
		Number:       0,
		PreviousHash: previousHash[:],
		Data:         configBytes,
		Config:       config,
		Timestamp:    time.Now(),
	}

	return block, nil
}

// CreateChannel 채널 생성
func (cm *ChannelManager) CreateChannel(channelID string, consortium string, ordererAddress string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 이미 존재하는 채널인지 확인
	if _, exists := cm.channels[channelID]; exists {
		return fmt.Errorf("channel %s already exists", channelID)
	}

	// 채널 구성 정보 생성
	config, err := cm.CreateChannelConfig(channelID, consortium, ordererAddress)
	if err != nil {
		return fmt.Errorf("failed to create channel config: %v", err)
	}

	// 제네시스 블록 생성
	genesisBlock, err := cm.CreateGenesisBlock(config)
	if err != nil {
		return fmt.Errorf("failed to create genesis block: %v", err)
	}

	// MSP 설정 생성
	channelConfig := &msp.MSPConfig{
		Name: fmt.Sprintf("%s.%s", "Org1MSP", channelID),
		CryptoConfig: &msp.FabricCryptoConfig{
			SignatureHashFamily:            "SHA2",
			IdentityIdentifierHashFunction: "SHA256",
		},
	}

	// MSP 인스턴스 생성 및 설정
	channelMSP := msp.NewFabricMSP()
	channelMSP.Setup(channelConfig)

	// 채널 생성
	channel := &Channel{
		Name:         channelID,
		Config:       config,
		GenesisBlock: genesisBlock,
		State:        make(map[string][]byte),
		MSP:          channelMSP,
		MSPConfig:    channelConfig, // MSP 설정 저장
	}

	// 채널 저장
	cm.channels[channelID] = channel

	// 파일 시스템에 채널 정보 저장
	if err := cm.saveChannel(channel); err != nil {
		// 저장 실패 시 메모리에서도 제거
		delete(cm.channels, channelID)
		return fmt.Errorf("failed to save channel: %v", err)
	}

	return nil
}

// GetChannel 채널 조회
func (cm *ChannelManager) GetChannel(channelID string) (*Channel, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	channel, exists := cm.channels[channelID]
	if !exists {
		return nil, fmt.Errorf("channel %s not found", channelID)
	}

	return channel, nil
}

// ListChannels 채널 목록 조회
func (cm *ChannelManager) ListChannels() []string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	channels := make([]string, 0, len(cm.channels))
	for channelID := range cm.channels {
		channels = append(channels, channelID)
	}

	return channels
}
