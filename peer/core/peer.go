package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
	"github.com/ddr4869/minifab/common/types"
	"github.com/ddr4869/minifab/peer"
	"github.com/ddr4869/minifab/peer/client"
	"github.com/pkg/errors"
)

type Peer struct {
	ID             string
	channelManager peer.ChannelManager
	transactions   []*types.Transaction
	mutex          sync.RWMutex
	chaincodePath  string
	msp            msp.MSP
	mspID          string
}

func NewPeer(id string, chaincodePath string, mspID string) *Peer {
	// MSP 인스턴스 생성
	fabricMSP := msp.NewFabricMSP()

	// 기본 MSP 설정
	config := &msp.MSPConfig{
		Name: mspID,
		CryptoConfig: &msp.FabricCryptoConfig{
			SignatureHashFamily:            "SHA2",
			IdentityIdentifierHashFunction: "SHA256",
		},
		NodeOUs: &msp.FabricNodeOUs{
			Enable: true,
			PeerOUIdentifier: &msp.FabricOUIdentifier{
				OrganizationalUnitIdentifier: "peer",
			},
		},
	}

	fabricMSP.Setup(config)

	return &Peer{
		ID:             id,
		channelManager: nil, // 나중에 manager 패키지에서 생성된 인스턴스로 설정
		transactions:   make([]*types.Transaction, 0),
		chaincodePath:  chaincodePath,
		msp:            fabricMSP,
		mspID:          mspID,
	}
}

// SetChannelManager 채널 매니저 설정 (의존성 주입)
func (p *Peer) SetChannelManager(cm peer.ChannelManager) {
	p.channelManager = cm
}

// NewPeerWithMSPFiles fabric-ca로 생성된 MSP 파일들을 사용하여 Peer 생성
func NewPeerWithMSPFiles(id string, chaincodePath string, mspID string, mspPath string) *Peer {
	// MSP 파일들로부터 MSP, Identity, PrivateKey 로드
	fabricMSP, identity, privateKey, err := msp.CreateMSPFromFiles(mspPath, mspID)
	if err != nil {
		logger.Errorf("Failed to create MSP from files: %v", err)
		// 실패 시 기본 MSP 사용
		return NewPeer(id, chaincodePath, mspID)
	}

	logger.Infof("✅ Successfully loaded MSP from %s", mspPath)
	logger.Info("📋 Identity Details:")
	logger.Infof("   - ID: %s", identity.GetIdentifier().Id)
	logger.Infof("   - MSP ID: %s", identity.GetMSPIdentifier())

	// 조직 단위 정보 출력
	ous := identity.GetOrganizationalUnits()
	if len(ous) > 0 {
		logger.Info("   - Organizational Units:")
		for _, ou := range ous {
			logger.Infof("     * %s", ou.OrganizationalUnitIdentifier)
		}
	}

	// privateKey는 나중에 사용할 수 있도록 저장 (현재는 로그만 출력)
	if privateKey != nil {
		logger.Info("🔑 Private key loaded successfully")
	}

	return &Peer{
		ID:             id,
		channelManager: nil, // 나중에 설정
		transactions:   make([]*types.Transaction, 0),
		chaincodePath:  chaincodePath,
		msp:            fabricMSP,
		mspID:          mspID,
	}
}

// JoinChannel joins an existing channel - channel must already exist
func (p *Peer) JoinChannel(channelName string, ordererClient client.OrdererService) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	logger.Infof("[Peer] Joining channel: %s", channelName)

	if p.channelManager == nil {
		return errors.New("channel manager not initialized")
	}

	// 채널이 이미 존재하는지 확인
	_, err := p.channelManager.GetChannel(channelName)
	if err != nil {
		// 채널이 존재하지 않으면 에러 반환 (채널 생성 요청하지 않음)
		logger.Errorf("[Peer] Channel %s not found locally", channelName)
		return errors.Errorf("channel %s does not exist - please create the channel first", channelName)
	}

	// 채널 가져오기
	channel, err := p.channelManager.GetChannel(channelName)
	if err != nil {
		return errors.Wrap(err, "failed to get channel")
	}

	// 채널용 MSP 생성
	channelMSP := msp.NewFabricMSP()
	config := &msp.MSPConfig{
		Name: fmt.Sprintf("%s.%s", p.mspID, channelName),
		CryptoConfig: &msp.FabricCryptoConfig{
			SignatureHashFamily:            "SHA2",
			IdentityIdentifierHashFunction: "SHA256",
		},
	}
	channelMSP.Setup(config)
	channel.MSP = channelMSP

	logger.Infof("[Peer] Successfully joined channel: %s", channelName)
	return nil
}

// JoinChannelWithProfile joins an existing channel with specific profile configuration
func (p *Peer) JoinChannelWithProfile(channelName, profileName string, ordererClient client.OrdererService) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	logger.Infof("[Peer] Joining channel: %s with profile: %s", channelName, profileName)

	if p.channelManager == nil {
		return errors.New("channel manager not initialized")
	}

	// 채널이 이미 존재하는지 확인
	_, err := p.channelManager.GetChannel(channelName)
	if err != nil {
		// 채널이 존재하지 않으면 에러 반환 (채널 생성 요청하지 않음)
		logger.Errorf("[Peer] Channel %s not found locally", channelName)
		return errors.Errorf("channel %s does not exist - please create the channel first", channelName)
	}

	// 채널 가져오기
	channel, err := p.channelManager.GetChannel(channelName)
	if err != nil {
		return errors.Wrap(err, "failed to get channel")
	}

	// 채널용 MSP 생성
	channelMSP := msp.NewFabricMSP()
	config := &msp.MSPConfig{
		Name: fmt.Sprintf("%s.%s", p.mspID, channelName),
		CryptoConfig: &msp.FabricCryptoConfig{
			SignatureHashFamily:            "SHA2",
			IdentityIdentifierHashFunction: "SHA256",
		},
	}
	channelMSP.Setup(config)
	channel.MSP = channelMSP

	logger.Infof("[Peer] Successfully joined channel: %s with profile: %s", channelName, profileName)
	return nil
}

func (p *Peer) SubmitTransaction(channelID string, payload []byte) (*types.Transaction, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.channelManager == nil {
		return nil, errors.New("channel manager not initialized")
	}

	channel, err := p.channelManager.GetChannel(channelID)
	if err != nil {
		return nil, errors.Errorf("channel %s not found", channelID)
	}

	// 임시 Identity와 서명 생성 (실제로는 인증서와 개인키 사용)
	identity := []byte(fmt.Sprintf("peer:%s:%s", p.mspID, p.ID))
	signature := []byte("temp_signature")

	tx := &types.Transaction{
		ID:        generateTransactionID(),
		ChannelID: channelID,
		Payload:   payload,
		Timestamp: time.Now(),
		Identity:  identity,
		Signature: signature,
	}

	p.transactions = append(p.transactions, tx)
	channel.Transactions = append(channel.Transactions, tx)

	return tx, nil
}

// ValidateTransaction 트랜잭션 검증 (MSP 사용)
func (p *Peer) ValidateTransaction(tx *types.Transaction) error {
	if p.channelManager == nil {
		return errors.New("channel manager not initialized")
	}

	channel, err := p.channelManager.GetChannel(tx.ChannelID)
	if err != nil {
		return errors.Errorf("channel %s not found", tx.ChannelID)
	}

	fmt.Println("channel", channel)
	// Identity 역직렬화
	identity, err := channel.MSP.DeserializeIdentity(tx.Identity)
	fmt.Println("identity", identity)
	if err != nil {
		return errors.Errorf("failed to deserialize identity: %v", err)
	}

	// Identity 검증
	if err := channel.MSP.ValidateIdentity(identity); err != nil {
		return errors.Errorf("invalid identity: %v", err)
	}

	// 서명 검증
	if err := identity.Verify(tx.Payload, tx.Signature); err != nil {
		return errors.Errorf("signature verification failed: %v", err)
	}

	return nil
}

// GetMSP MSP 인스턴스 반환
func (p *Peer) GetMSP() msp.MSP {
	return p.msp
}

// GetMSPID MSP ID 반환
func (p *Peer) GetMSPID() string {
	return p.mspID
}

// CreateChannel creates a channel via orderer and then creates it locally
func (p *Peer) CreateChannel(channelName string, ordererClient client.OrdererService) error {
	logger.Infof("[Peer] Creating channel: %s", channelName)

	// 1. First, request channel creation from orderer
	if ordererClient == nil {
		return errors.New("orderer client is required for channel creation")
	}

	if err := ordererClient.CreateChannel(channelName); err != nil {
		return errors.Wrapf(err, "failed to create channel %s via orderer", channelName)
	}

	// 2. Then create the channel locally
	if p.channelManager == nil {
		return errors.New("channel manager not initialized")
	}

	if err := p.channelManager.CreateChannel(channelName, "SampleConsortium", "localhost:7050"); err != nil {
		return errors.Wrapf(err, "failed to create local channel %s", channelName)
	}

	logger.Infof("[Peer] Channel %s created successfully", channelName)
	return nil
}

// CreateChannelWithProfile creates a channel with specific profile via orderer and then creates it locally
func (p *Peer) CreateChannelWithProfile(channelName, profileName string, ordererClient client.OrdererService) error {
	logger.Infof("[Peer] Creating channel: %s with profile: %s", channelName, profileName)

	// 1. First, request channel creation from orderer with profile
	if ordererClient == nil {
		return errors.New("orderer client is required for channel creation")
	}

	if err := ordererClient.CreateChannelWithProfile(channelName, profileName, "config/configtx.yaml"); err != nil {
		return errors.Wrapf(err, "failed to create channel %s via orderer with profile %s", channelName, profileName)
	}

	// 2. Then create the channel locally
	if p.channelManager == nil {
		return errors.New("channel manager not initialized")
	}

	if err := p.channelManager.CreateChannel(channelName, "SampleConsortium", "localhost:7050"); err != nil {
		return errors.Wrapf(err, "failed to create local channel %s", channelName)
	}

	logger.Infof("[Peer] Channel %s created successfully with profile %s", channelName, profileName)
	return nil
}

// GetChannelManager 채널 관리자 반환
func (p *Peer) GetChannelManager() peer.ChannelManager {
	return p.channelManager
}

func generateTransactionID() string {
	// 실제 구현에서는 고유한 트랜잭션 ID 생성 로직 구현
	return fmt.Sprintf("tx_%d", time.Now().UnixNano())
}
