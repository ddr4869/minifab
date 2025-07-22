package channel

import (
	"fmt"
	"sync"
	"time"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
	"github.com/ddr4869/minifab/common/types"
	"github.com/ddr4869/minifab/peer/client"
	"github.com/pkg/errors"
)

var (
	// 전역 peer 인스턴스 관리
	globalPeer  PeerInterface
	globalMutex sync.RWMutex
)

// PeerInterface는 channel 패키지가 peer와 상호작용하기 위한 인터페이스입니다
type PeerInterface interface {
	GetID() string
	GetMSPID() string
	GetOrdererClient() client.OrdererService
	GetChannelManager() ChannelManager
	SetChannelManager(cm ChannelManager)
}

// ChannelManager interface to break circular dependency
type ChannelManager interface {
	CreateChannel(channelID string, consortium string, ordererAddress string) error
	GetChannel(channelID string) (*types.Channel, error)
	ListChannels() []string
	GetChannelNames() []string
}

// SetGlobalPeer sets the global peer instance for channel operations
func SetGlobalPeer(peer PeerInterface) {
	globalMutex.Lock()
	defer globalMutex.Unlock()
	globalPeer = peer
}

// getGlobalPeer returns the global peer instance
func getGlobalPeer() PeerInterface {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	return globalPeer
}

// ensureChannelManagerInitialized 채널 매니저가 초기화되지 않았으면 lazy initialization을 수행합니다
func ensureChannelManagerInitialized(peer PeerInterface) error {
	if peer.GetChannelManager() != nil {
		return nil // 이미 초기화됨
	}

	// 채널 매니저 생성 및 설정
	channelManager := NewManager()
	peer.SetChannelManager(channelManager)

	logger.Infof("✅ Channel manager lazy initialized for peer %s", peer.GetID())
	return nil
}

// CreateChannel creates a channel via orderer and then creates it locally
func CreateChannel(channelName string) error {
	return CreateChannelWithProfile(channelName, "testchannel0")
}

// CreateChannelWithProfile creates a channel with specific profile via orderer and then creates it locally
func CreateChannelWithProfile(channelName, profileName string) error {
	peer := getGlobalPeer()
	if peer == nil {
		return errors.New("peer not initialized")
	}

	logger.Infof("[Channel] Creating channel: %s with profile: %s", channelName, profileName)

	// 1. First, request channel creation from orderer with profile
	ordererClient := peer.GetOrdererClient()
	if ordererClient == nil {
		return errors.New("orderer client not initialized")
	}

	if err := ordererClient.CreateChannelWithProfile(channelName, profileName, "config/configtx.yaml"); err != nil {
		return errors.Wrapf(err, "failed to create channel %s via orderer with profile %s", channelName, profileName)
	}

	// 2. Then create the channel locally
	// Lazy initialization of channel manager
	if err := ensureChannelManagerInitialized(peer); err != nil {
		return errors.Wrap(err, "failed to initialize channel manager")
	}

	if err := peer.GetChannelManager().CreateChannel(channelName, "SampleConsortium", "localhost:7050"); err != nil {
		return errors.Wrapf(err, "failed to create local channel %s", channelName)
	}

	logger.Infof("[Channel] Channel %s created successfully with profile %s", channelName, profileName)
	return nil
}

// JoinChannel joins an existing channel - channel must already exist
func JoinChannel(channelName string) error {
	return JoinChannelWithProfile(channelName, "")
}

// JoinChannelWithProfile joins an existing channel with specific profile configuration
func JoinChannelWithProfile(channelName, profileName string) error {
	peer := getGlobalPeer()
	if peer == nil {
		return errors.New("peer not initialized")
	}

	if profileName == "" {
		logger.Infof("[Channel] Joining channel: %s", channelName)
	} else {
		logger.Infof("[Channel] Joining channel: %s with profile: %s", channelName, profileName)
	}

	// Lazy initialization of channel manager
	if err := ensureChannelManagerInitialized(peer); err != nil {
		return errors.Wrap(err, "failed to initialize channel manager")
	}

	ordererClient := peer.GetOrdererClient()
	if ordererClient == nil {
		return errors.New("orderer client not initialized")
	}

	// 채널이 이미 존재하는지 확인
	_, err := peer.GetChannelManager().GetChannel(channelName)
	if err != nil {
		// 채널이 존재하지 않으면 에러 반환 (채널 생성 요청하지 않음)
		logger.Errorf("[Channel] Channel %s not found locally", channelName)
		return errors.Errorf("channel %s does not exist - please create the channel first", channelName)
	}

	// 채널 가져오기
	channel, err := peer.GetChannelManager().GetChannel(channelName)
	if err != nil {
		return errors.Wrap(err, "failed to get channel")
	}

	// 채널용 MSP 생성
	channelMSP := msp.NewFabricMSP()
	config := &msp.MSPConfig{
		Name: fmt.Sprintf("%s.%s", peer.GetMSPID(), channelName),
		CryptoConfig: &msp.FabricCryptoConfig{
			SignatureHashFamily:            "SHA2",
			IdentityIdentifierHashFunction: "SHA256",
		},
	}
	channelMSP.Setup(config)
	channel.MSP = channelMSP

	if profileName == "" {
		logger.Infof("[Channel] Successfully joined channel: %s", channelName)
	} else {
		logger.Infof("[Channel] Successfully joined channel: %s with profile: %s", channelName, profileName)
	}
	return nil
}

// ListChannels lists all available channels
func ListChannels() error {
	peer := getGlobalPeer()
	if peer == nil {
		return errors.New("peer not initialized")
	}

	// Lazy initialization of channel manager
	if err := ensureChannelManagerInitialized(peer); err != nil {
		return errors.Wrap(err, "failed to initialize channel manager")
	}

	channels := peer.GetChannelManager().ListChannels()
	if len(channels) == 0 {
		logger.Info("No channels found")
		return nil
	}
	logger.Info("Available channels:")
	for _, channel := range channels {
		logger.Infof("- %s", channel)
	}
	return nil
}

// SubmitTransaction submits a transaction to a channel
func SubmitTransaction(channelID string, payload []byte) (*types.Transaction, error) {
	peer := getGlobalPeer()
	if peer == nil {
		return nil, errors.New("peer not initialized")
	}

	// Lazy initialization of channel manager
	if err := ensureChannelManagerInitialized(peer); err != nil {
		return nil, errors.Wrap(err, "failed to initialize channel manager")
	}

	ordererClient := peer.GetOrdererClient()
	if ordererClient == nil {
		return nil, errors.New("orderer client not initialized")
	}

	channel, err := peer.GetChannelManager().GetChannel(channelID)
	if err != nil {
		return nil, errors.Errorf("channel %s not found", channelID)
	}

	// 임시 Identity와 서명 생성 (실제로는 인증서와 개인키 사용)
	identity := []byte(fmt.Sprintf("peer:%s:%s", peer.GetMSPID(), peer.GetID()))
	signature := []byte("temp_signature")

	tx := &types.Transaction{
		ID:        generateTransactionID(),
		ChannelID: channelID,
		Payload:   payload,
		Timestamp: time.Now(),
		Identity:  identity,
		Signature: signature,
	}

	// Orderer에 트랜잭션 제출
	if err := ordererClient.SubmitTransaction(tx); err != nil {
		return nil, errors.Wrap(err, "failed to submit transaction to orderer")
	}

	channel.Transactions = append(channel.Transactions, tx)

	logger.Infof("Transaction submitted successfully: %s", tx.ID)
	return tx, nil
}

func generateTransactionID() string {
	// 실제 구현에서는 고유한 트랜잭션 ID 생성 로직 구현
	return fmt.Sprintf("tx_%d", time.Now().UnixNano())
}
