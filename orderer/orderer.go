package orderer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ddr4869/minifab/common/common"
	"github.com/ddr4869/minifab/common/configtx"
	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
	pb_common "github.com/ddr4869/minifab/proto/common"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

const (
	// File permissions
	GenesisFilePermissions = 0644

	// Hash algorithms
	DefaultHashFamily   = "SHA2"
	DefaultHashFunction = "SHA256"

	// Organizational units
	DefaultOrdererOU = "orderer"

	// Default MSP ID
	DefaultMSPID = "DefaultOrdererMSP"

	// Block validation
	MinBlockDataSize = 1
	MaxBlockDataSize = 32 * 1024 * 1024 // 32MB

	// Channel validation
	MaxChannelNameLength = 249
	MinChannelNameLength = 1
)

type Orderer struct {
	blocks         []*pb_common.Block
	currentBlock   *pb_common.Block
	mutex          sync.RWMutex
	channels       map[string]*common.Channel
	msp            msp.MSP
	mspID          string
	systemChannel  string
	isBootstrapped bool
}

// NewOrdererWithMSPFiles fabric-ca로 생성된 MSP 파일들을 사용하여 Orderer 생성
func NewOrderer(mspID string, mspPath string) (*Orderer, error) {
	// MSP 파일들로부터 MSP, Identity, PrivateKey 로드
	fabricMSP, err := msp.CreateMSPFromFiles(mspID, mspPath)
	if err != nil {
		logger.Errorf("Failed to create MSP from files: %v", err)
		return nil, err
	}

	logger.Infof("✅ Successfully loaded Orderer MSP from %s", mspPath)
	logger.Info("📋 Orderer Identity Details:")
	logger.Infof("   - ID: %s", fabricMSP.GetIdentifier().Id)
	logger.Infof("   - MSP ID: %s", fabricMSP.GetIdentifier().Mspid)

	// 조직 단위 정보 출력
	// ous := identity.GetOrganizationalUnits()
	// if len(ous) > 0 {
	// 	logger.Info("   - Organizational Units:")
	// 	for _, ou := range ous {
	// 		logger.Infof("     * %s", ou.OrganizationalUnitIdentifier)
	// 	}
	// }

	return &Orderer{
		blocks:   make([]*pb_common.Block, 0),
		channels: make(map[string]*common.Channel),
		msp:      fabricMSP,
		mspID:    mspID,
	}, nil
}

func (o *Orderer) CreateBlock(data []byte) (*pb_common.Block, error) {
	if len(data) < MinBlockDataSize {
		return nil, errors.New("block data cannot be empty")
	}

	if len(data) > MaxBlockDataSize {
		return nil, errors.Errorf("block data size %d exceeds maximum allowed size %d", len(data), MaxBlockDataSize)
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	block := &pb_common.Block{
		Header: &pb_common.BlockHeader{
			Number:       uint64(len(o.blocks)),
			PreviousHash: o.getLastBlockHash(),
			HeaderType:   pb_common.BlockType_BLOCK_TYPE_CONFIG,
		},
		Data: &pb_common.BlockData{
			Transactions: [][]byte{data},
		},
	}

	o.blocks = append(o.blocks, block)
	o.currentBlock = block

	return block, nil
}

func (o *Orderer) getLastBlockHash() []byte {
	if len(o.blocks) == 0 {
		return nil
	}

	lastBlock := o.blocks[len(o.blocks)-1]
	return o.calculateBlockHash(lastBlock)
}

// calculateBlockHash calculates the hash of a block
func (o *Orderer) calculateBlockHash(block *pb_common.Block) []byte {
	if block == nil {
		return nil
	}

	// TODO: 블록 해시 계산 로직 추가
	hash := sha256.New()
	return hash.Sum(nil)
}

// GetMSP MSP 인스턴스 반환
func (o *Orderer) GetMSP() msp.MSP {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.msp
}

// GetMSPID MSP ID 반환
func (o *Orderer) GetMSPID() string {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.mspID
}

// GetBlockCount returns the total number of blocks
func (o *Orderer) GetBlockCount() uint64 {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return uint64(len(o.blocks))
}

// GetBlock returns a block by number
func (o *Orderer) GetBlock(blockNumber uint64) (*pb_common.Block, error) {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	if blockNumber >= uint64(len(o.blocks)) {
		return nil, errors.Errorf("block %d not found", blockNumber)
	}

	return o.blocks[blockNumber], nil
}

// GetChannels returns a list of all channel names
func (o *Orderer) GetChannels() []string {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	channels := make([]string, 0, len(o.channels))
	for name := range o.channels {
		channels = append(channels, name)
	}
	return channels
}

// GetChannel returns a channel by name
func (o *Orderer) GetChannel(channelName string) (*common.Channel, error) {
	if channelName == "" {
		return nil, errors.New("channel name cannot be empty")
	}

	o.mutex.RLock()
	defer o.mutex.RUnlock()

	channel, exists := o.channels[channelName]
	if !exists {
		return nil, errors.Errorf("channel %s not found", channelName)
	}

	return channel, nil
}

// BootstrapNetwork 네트워크 부트스트랩 (제네시스 블록 생성)
func (o *Orderer) BootstrapNetwork(genesisConfig *configtx.SystemChannelInfo) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if err := o.validateBootstrapPreconditions(genesisConfig); err != nil {
		return err
	}

	logger.Info("Starting network bootstrap process")

	err := o.generateGenesisBlock(genesisConfig)
	if err != nil {
		return errors.Wrap(err, "failed to generate genesis block")
	}

	o.isBootstrapped = true
	return nil
}

func (o *Orderer) validateBootstrapPreconditions(genesisConfig *configtx.SystemChannelInfo) error {
	if o.isBootstrapped {
		return errors.New("network is already bootstrapped")
	}

	if genesisConfig == nil {
		return errors.New("genesis config cannot be nil")
	}

	return nil
}

func (o *Orderer) generateGenesisBlock(genesisConfig *configtx.SystemChannelInfo) error {
	// 설정 트랜잭션 데이터 직렬화
	configTxData, err := json.Marshal(genesisConfig)
	if err != nil {
		return errors.Wrap(err, "failed to marshal genesis config")
	}

	// 블록 헤더 생성
	header := &pb_common.BlockHeader{
		Number:       0,   // 제네시스 블록은 항상 0
		PreviousHash: nil, // 제네시스 블록은 이전 해시가 없음
		HeaderType:   pb_common.BlockType_BLOCK_TYPE_CONFIG,
	}

	// 블록 데이터 생성
	blockData := &pb_common.BlockData{
		Transactions: [][]byte{
			configTxData, // 설정 트랜잭션 데이터
		},
	}

	// 블록 메타데이터 생성
	metadata := &pb_common.BlockMetadata{
		// CreatorCertificate: o.msp.GetIdentifier().Id,
		CreatorSignature: []byte{},  // 실제 서명 로직 필요
		ValidationBitmap: []byte{1}, // 제네시스 블록은 항상 유효
		AccumulatedHash:  []byte{},  // 제네시스 블록은 빈 해시
	}

	// 블록 생성
	block := &pb_common.Block{
		Header:   header,
		Data:     blockData,
		Metadata: metadata,
	}

	// 현재 블록 해시 계산
	blockHash := o.calculateBlockHash(block)
	header.CurrentBlockHash = blockHash

	// 제네시스 블록 구조체 생성
	genesisBlock := &pb_common.GenesisBlock{
		Block:       block,
		ChannelId:   "SYSTEM_CHANNEL",
		StoredAt:    time.Now().Format(time.RFC3339),
		IsCommitted: true,
		BlockHash:   fmt.Sprintf("%x", blockHash),
	}

	// protobuf로 직렬화
	protoData, err := proto.Marshal(genesisBlock)
	if err != nil {
		return errors.Wrap(err, "failed to marshal genesis block")
	}

	// 파일에 저장 (protobuf 바이너리 형태)
	if err := os.WriteFile("./blocks/genesis.block", protoData, GenesisFilePermissions); err != nil {
		return errors.Wrap(err, "failed to write genesis block file")
	}

	// JSON 형태로도 저장 (디버깅용)
	jsonData, err := json.MarshalIndent(genesisBlock, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal genesis block to JSON")
	}

	if err := os.WriteFile("genesis.json", jsonData, GenesisFilePermissions); err != nil {
		return errors.Wrap(err, "failed to write genesis JSON file")
	}

	logger.Info("Genesis block created and saved successfully")
	return nil
}
