package core

import (
	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
	"github.com/ddr4869/minifab/peer/common"
)

var (
	OrdererAddress string
	PeerID         string
	ChaincodePath  string
	MspID          string
	MspPath        string
)

type Peer struct {
	// ID             string
	// channelManager peer.ChannelManager
	// transactions   []*types.Transaction
	// mutex          sync.RWMutex
	// chaincodePath  string
	// msp            msp.MSP
	// mspID          string
	// ordererClient  client.OrdererService // OrdererService 필드 추가
	PeerConfig    *PeerConfig
	OrdererClient *common.OrdererClient
}

func NewPeer(mspId, mspPath, ordererAddress string) (*Peer, error) {
	// MSP 파일들로부터 MSP, Identity, PrivateKey 로드
	fabricMSP, identity, privateKey, err := msp.CreateMSPFromFiles(MspPath, MspID)
	if err != nil {
		logger.Errorf("Failed to create MSP from files: %v", err)
		return nil, err
	}

	logger.Infof("✅ Successfully loaded MSP from %s", mspPath)
	logger.Info("📋 Identity Details:")
	logger.Infof("   - ID: %s", identity.GetIdentifier().Id)
	logger.Infof("   - MSP ID: %s", identity.GetIdentifier().Mspid)

	// 조직 단위 정보 출력
	// ous := identity.GetOrganizationalUnits()
	// if len(ous) > 0 {
	// 	logger.Info("   - Organizational Units:")
	// 	for _, ou := range ous {
	// 		logger.Infof("     * %s", ou.OrganizationalUnitIdentifier)
	// 	}
	// }

	// privateKey는 나중에 사용할 수 있도록 저장 (현재는 로그만 출력)
	if privateKey != nil {
		logger.Info("🔑 Private key loaded successfully")
	}

	ordererClient, err := common.NewOrdererClient(ordererAddress)
	if err != nil {
		logger.Errorf("Failed to create orderer client: %v", err)
		return nil
	}

	return &Peer{
		PeerConfig: &PeerConfig{
			PeerID: PeerID,
			Msp:    fabricMSP,
		},
		OrdererClient: ordererClient,
	}
}

// // SetChannelManager 채널 매니저 설정 (의존성 주입)
// func (p *Peer) SetChannelManager(cm peer.ChannelManager) {
// 	p.channelManager = cm
// }

// // ensureChannelManagerInitialized 채널 매니저가 초기화되지 않았으면 lazy initialization을 수행합니다
// func (p *Peer) ensureChannelManagerInitialized() error {
// 	if p.channelManager != nil {
// 		return nil // 이미 초기화됨
// 	}

// 	// 채널 매니저 생성을 위해 간단한 in-memory 구현
// 	// 실제로는 channel.NewManager()를 사용하고 싶지만 import cycle 때문에 불가능
// 	// 대신 기본적인 채널 맵을 가진 구조체를 생성
// 	channelManager := &simpleChannelManager{
// 		channels: make(map[string]*types.Channel),
// 	}
// 	p.SetChannelManager(channelManager)

// 	logger.Infof("✅ Channel manager lazy initialized for peer %s", p.ID)
// 	return nil
// }

// // simpleChannelManager는 간단한 채널 매니저 구현체입니다 (import cycle 방지용)
// type simpleChannelManager struct {
// 	channels map[string]*types.Channel
// 	mutex    sync.RWMutex
// }

// func (scm *simpleChannelManager) CreateChannel(channelID string, consortium string, ordererAddress string) error {
// 	scm.mutex.Lock()
// 	defer scm.mutex.Unlock()

// 	if _, exists := scm.channels[channelID]; exists {
// 		return nil // 이미 존재함
// 	}

// 	channel := &types.Channel{
// 		Name: channelID,
// 		Config: &types.ChannelConfig{
// 			ChannelID:      channelID,
// 			Consortium:     consortium,
// 			OrdererAddress: ordererAddress,
// 		},
// 		Transactions: make([]*types.Transaction, 0),
// 		State:        make(map[string][]byte),
// 	}

// 	scm.channels[channelID] = channel
// 	logger.Infof("Channel %s created locally", channelID)
// 	return nil
// }

// func (scm *simpleChannelManager) GetChannel(channelID string) (*types.Channel, error) {
// 	scm.mutex.RLock()
// 	defer scm.mutex.RUnlock()

// 	channel, exists := scm.channels[channelID]
// 	if !exists {
// 		return nil, errors.Errorf("channel %s not found", channelID)
// 	}

// 	return channel, nil
// }

// func (scm *simpleChannelManager) ListChannels() []string {
// 	scm.mutex.RLock()
// 	defer scm.mutex.RUnlock()

// 	channels := make([]string, 0, len(scm.channels))
// 	for name := range scm.channels {
// 		channels = append(channels, name)
// 	}

// 	return channels
// }

// func (scm *simpleChannelManager) GetChannelNames() []string {
// 	return scm.ListChannels()
// }

// // JoinChannel joins an existing channel - channel must already exist
// func (p *Peer) JoinChannel(channelName string) error {
// 	// Note: This will be implemented via the channel package operations
// 	// For now, keep the original implementation to avoid import cycles
// 	p.mutex.Lock()
// 	defer p.mutex.Unlock()

// 	logger.Infof("[Peer] Joining channel: %s", channelName)

// 	// Lazy initialization of channel manager
// 	if err := p.ensureChannelManagerInitialized(); err != nil {
// 		return errors.Wrap(err, "failed to initialize channel manager")
// 	}

// 	if p.ordererClient == nil {
// 		return errors.New("orderer client not initialized")
// 	}

// 	// 채널이 이미 존재하는지 확인
// 	_, err := p.channelManager.GetChannel(channelName)
// 	if err != nil {
// 		// 채널이 존재하지 않으면 에러 반환 (채널 생성 요청하지 않음)
// 		logger.Errorf("[Peer] Channel %s not found locally", channelName)
// 		return errors.Errorf("channel %s does not exist - please create the channel first", channelName)
// 	}

// 	// 채널 가져오기
// 	channel, err := p.channelManager.GetChannel(channelName)
// 	if err != nil {
// 		return errors.Wrap(err, "failed to get channel")
// 	}

// 	// 채널용 MSP 생성
// 	channelMSP := msp.NewFabricMSP()
// 	config := &msp.MSPConfig{
// 		Name: fmt.Sprintf("%s.%s", p.mspID, channelName),
// 		CryptoConfig: &msp.FabricCryptoConfig{
// 			SignatureHashFamily:            "SHA2",
// 			IdentityIdentifierHashFunction: "SHA256",
// 		},
// 	}
// 	channelMSP.Setup(config)
// 	channel.MSP = channelMSP

// 	logger.Infof("[Peer] Successfully joined channel: %s", channelName)
// 	return nil
// }

// // JoinChannelWithProfile joins an existing channel with specific profile configuration
// func (p *Peer) JoinChannelWithProfile(channelName, profileName string) error {
// 	p.mutex.Lock()
// 	defer p.mutex.Unlock()

// 	logger.Infof("[Peer] Joining channel: %s with profile: %s", channelName, profileName)

// 	// Lazy initialization of channel manager
// 	if err := p.ensureChannelManagerInitialized(); err != nil {
// 		return errors.Wrap(err, "failed to initialize channel manager")
// 	}

// 	if p.ordererClient == nil {
// 		return errors.New("orderer client not initialized")
// 	}

// 	// 채널이 이미 존재하는지 확인
// 	_, err := p.channelManager.GetChannel(channelName)
// 	if err != nil {
// 		// 채널이 존재하지 않으면 에러 반환 (채널 생성 요청하지 않음)
// 		logger.Errorf("[Peer] Channel %s not found locally", channelName)
// 		return errors.Errorf("channel %s does not exist - please create the channel first", channelName)
// 	}

// 	// 채널 가져오기
// 	channel, err := p.channelManager.GetChannel(channelName)
// 	if err != nil {
// 		return errors.Wrap(err, "failed to get channel")
// 	}

// 	// 채널용 MSP 생성
// 	channelMSP := msp.NewFabricMSP()
// 	config := &msp.MSPConfig{
// 		Name: fmt.Sprintf("%s.%s", p.mspID, channelName),
// 		CryptoConfig: &msp.FabricCryptoConfig{
// 			SignatureHashFamily:            "SHA2",
// 			IdentityIdentifierHashFunction: "SHA256",
// 		},
// 	}
// 	channelMSP.Setup(config)
// 	channel.MSP = channelMSP

// 	logger.Infof("[Peer] Successfully joined channel: %s with profile: %s", channelName, profileName)
// 	return nil
// }

// func (p *Peer) SubmitTransaction(channelID string, payload []byte) (*types.Transaction, error) {
// 	p.mutex.Lock()
// 	defer p.mutex.Unlock()

// 	// Lazy initialization of channel manager
// 	if err := p.ensureChannelManagerInitialized(); err != nil {
// 		return nil, errors.Wrap(err, "failed to initialize channel manager")
// 	}

// 	if p.ordererClient == nil {
// 		return nil, errors.New("orderer client not initialized")
// 	}

// 	channel, err := p.channelManager.GetChannel(channelID)
// 	if err != nil {
// 		return nil, errors.Errorf("channel %s not found", channelID)
// 	}

// 	// 임시 Identity와 서명 생성 (실제로는 인증서와 개인키 사용)
// 	identity := []byte(fmt.Sprintf("peer:%s:%s", p.mspID, p.ID))
// 	signature := []byte("temp_signature")

// 	tx := &types.Transaction{
// 		ID:        generateTransactionID(),
// 		ChannelID: channelID,
// 		Payload:   payload,
// 		Timestamp: time.Now(),
// 		Identity:  identity,
// 		Signature: signature,
// 	}

// 	// Orderer에 트랜잭션 제출
// 	if err := p.ordererClient.SubmitTransaction(tx); err != nil {
// 		return nil, errors.Wrap(err, "failed to submit transaction to orderer")
// 	}

// 	p.transactions = append(p.transactions, tx)
// 	channel.Transactions = append(channel.Transactions, tx)

// 	logger.Infof("Transaction submitted successfully: %s", tx.ID)
// 	return tx, nil
// }

// // ValidateTransaction 트랜잭션 검증 (MSP 사용)
// func (p *Peer) ValidateTransaction(tx *types.Transaction) error {
// 	if p.channelManager == nil {
// 		return errors.New("channel manager not initialized")
// 	}

// 	channel, err := p.channelManager.GetChannel(tx.ChannelID)
// 	if err != nil {
// 		return errors.Errorf("channel %s not found", tx.ChannelID)
// 	}

// 	fmt.Println("channel", channel)
// 	// Identity 역직렬화
// 	identity, err := channel.MSP.DeserializeIdentity(tx.Identity)
// 	fmt.Println("identity", identity)
// 	if err != nil {
// 		return errors.Errorf("failed to deserialize identity: %v", err)
// 	}

// 	// Identity 검증
// 	if err := channel.MSP.ValidateIdentity(identity); err != nil {
// 		return errors.Errorf("invalid identity: %v", err)
// 	}

// 	// 서명 검증
// 	if err := identity.Verify(tx.Payload, tx.Signature); err != nil {
// 		return errors.Errorf("signature verification failed: %v", err)
// 	}

// 	return nil
// }

// // GetMSP MSP 인스턴스 반환
// func (p *Peer) GetMSP() msp.MSP {
// 	return p.msp
// }

// // GetMSPID MSP ID 반환
// func (p *Peer) GetMSPID() string {
// 	return p.mspID
// }

// // CreateChannel creates a channel via orderer and then creates it locally
// func (p *Peer) CreateChannel(channelName string) error {
// 	logger.Infof("[Peer] Creating channel: %s", channelName)

// 	// 1. First, request channel creation from orderer
// 	if p.ordererClient == nil {
// 		return errors.New("orderer client not initialized")
// 	}

// 	if err := p.ordererClient.CreateChannel(channelName); err != nil {
// 		return errors.Wrapf(err, "failed to create channel %s via orderer", channelName)
// 	}

// 	// 2. Then create the channel locally
// 	// Lazy initialization of channel manager
// 	if err := p.ensureChannelManagerInitialized(); err != nil {
// 		return errors.Wrap(err, "failed to initialize channel manager")
// 	}

// 	if err := p.channelManager.CreateChannel(channelName, "SampleConsortium", "localhost:7050"); err != nil {
// 		return errors.Wrapf(err, "failed to create local channel %s", channelName)
// 	}

// 	logger.Infof("[Peer] Channel %s created successfully", channelName)
// 	return nil
// }

// // CreateChannelWithProfile creates a channel with specific profile via orderer and then creates it locally
// func (p *Peer) CreateChannelWithProfile(channelName, profileName string) error {
// 	logger.Infof("[Peer] Creating channel: %s with profile: %s", channelName, profileName)

// 	// 1. First, request channel creation from orderer with profile
// 	if p.ordererClient == nil {
// 		return errors.New("orderer client not initialized")
// 	}

// 	if err := p.ordererClient.CreateChannelWithProfile(channelName, profileName, "config/configtx.yaml"); err != nil {
// 		return errors.Wrapf(err, "failed to create channel %s via orderer with profile %s", channelName, profileName)
// 	}

// 	// 2. Then create the channel locally
// 	// Lazy initialization of channel manager
// 	if err := p.ensureChannelManagerInitialized(); err != nil {
// 		return errors.Wrap(err, "failed to initialize channel manager")
// 	}

// 	if err := p.channelManager.CreateChannel(channelName, "SampleConsortium", "localhost:7050"); err != nil {
// 		return errors.Wrapf(err, "failed to create local channel %s", channelName)
// 	}

// 	logger.Infof("[Peer] Channel %s created successfully with profile %s", channelName, profileName)
// 	return nil
// }

// // GetChannelManager 채널 관리자 반환
// func (p *Peer) GetChannelManager() peer.ChannelManager {
// 	// Lazy initialization of channel manager
// 	if err := p.ensureChannelManagerInitialized(); err != nil {
// 		logger.Errorf("Failed to initialize channel manager: %v", err)
// 		return nil
// 	}
// 	return p.channelManager
// }

// // GetOrdererClient returns the orderer client
// func (p *Peer) GetOrdererClient() client.OrdererService {
// 	return p.ordererClient
// }

// func generateTransactionID() string {
// 	// 실제 구현에서는 고유한 트랜잭션 ID 생성 로직 구현
// 	return fmt.Sprintf("tx_%d", time.Now().UnixNano())
// }
