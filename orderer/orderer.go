package orderer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
)

// OrdererInterface defines the interface for orderer operations
type OrdererInterface interface {
	CreateBlock(data []byte) (*Block, error)
	CreateChannel(channelName string) error
	ValidateTransaction(channelID string, serializedIdentity []byte, signature []byte, payload []byte) error
	GetMSP() msp.MSP
	GetMSPID() string
	GetBlockCount() uint64
	GetBlock(blockNumber uint64) (*Block, error)
	GetChannels() []string
	GetChannel(channelName string) (*Channel, error)
	BootstrapNetwork(genesisConfig *GenesisConfig) error
	IsBootstrapped() bool
	GetGenesisBlock() *GenesisBlock
	GetSystemChannel() string
	SaveGenesisBlock(filePath string) error
	LoadGenesisBlock(filePath string) error
}

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

type Block struct {
	Number       uint64
	PreviousHash []byte
	Data         []byte
	Timestamp    time.Time
}

type Orderer struct {
	blocks         []*Block
	currentBlock   *Block
	mutex          sync.RWMutex
	channels       map[string]*Channel
	msp            msp.MSP
	mspID          string
	genesisBlock   *GenesisBlock
	systemChannel  string
	isBootstrapped bool
}

type Channel struct {
	Name   string
	Blocks []*Block
	MSP    msp.MSP
}

func NewOrderer(mspID string) *Orderer {
	if mspID == "" {
		logger.Warn("Empty MSP ID provided, using default")
		mspID = "DefaultOrdererMSP"
	}

	// MSP 인스턴스 생성
	fabricMSP := msp.NewFabricMSP()

	// 기본 MSP 설정
	config := &msp.MSPConfig{
		Name: mspID,
		CryptoConfig: &msp.FabricCryptoConfig{
			SignatureHashFamily:            DefaultHashFamily,
			IdentityIdentifierHashFunction: DefaultHashFunction,
		},
		NodeOUs: &msp.FabricNodeOUs{
			Enable: true,
			OrdererOUIdentifier: &msp.FabricOUIdentifier{
				OrganizationalUnitIdentifier: DefaultOrdererOU,
			},
		},
	}

	if err := fabricMSP.Setup(config); err != nil {
		logger.Errorf("Failed to setup MSP: %v", err)
		// Continue with a basic MSP setup
	}

	return &Orderer{
		blocks:   make([]*Block, 0),
		channels: make(map[string]*Channel),
		msp:      fabricMSP,
		mspID:    mspID,
	}
}

// NewOrdererWithMSPFiles fabric-ca로 생성된 MSP 파일들을 사용하여 Orderer 생성
func NewOrdererWithMSPFiles(mspID string, mspPath string) *Orderer {
	// MSP 파일들로부터 MSP, Identity, PrivateKey 로드
	fabricMSP, identity, privateKey, err := msp.CreateMSPFromFiles(mspPath, mspID)
	if err != nil {
		logger.Errorf("Failed to create MSP from files: %v", err)
		// 실패 시 기본 MSP 사용
		return NewOrderer(mspID)
	}

	logger.Infof("✅ Successfully loaded Orderer MSP from %s", mspPath)
	logger.Info("📋 Orderer Identity Details:")
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
		logger.Info("🔑 Orderer private key loaded successfully")
	}

	return &Orderer{
		blocks:   make([]*Block, 0),
		channels: make(map[string]*Channel),
		msp:      fabricMSP,
		mspID:    mspID,
	}
}

func (o *Orderer) CreateBlock(data []byte) (*Block, error) {
	if len(data) < MinBlockDataSize {
		return nil, fmt.Errorf("block data cannot be empty")
	}

	if len(data) > MaxBlockDataSize {
		return nil, fmt.Errorf("block data size %d exceeds maximum allowed size %d", len(data), MaxBlockDataSize)
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	block := &Block{
		Number:       uint64(len(o.blocks)),
		PreviousHash: o.getLastBlockHash(),
		Data:         data,
		Timestamp:    time.Now(),
	}

	o.blocks = append(o.blocks, block)
	o.currentBlock = block

	logger.Debugf("Created block %d with %d bytes of data", block.Number, len(data))
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
func (o *Orderer) calculateBlockHash(block *Block) []byte {
	if block == nil {
		return nil
	}

	// Create a deterministic representation of the block for hashing
	blockData := fmt.Sprintf("%d:%x:%s",
		block.Number,
		block.PreviousHash,
		block.Timestamp.Format(time.RFC3339Nano))

	// Add block data if present
	if len(block.Data) > 0 {
		blockData += ":" + string(block.Data)
	}

	// Use SHA256 for hashing (consistent with Fabric)
	hash := sha256.Sum256([]byte(blockData))
	return hash[:]
}

func (o *Orderer) CreateChannel(channelName string) error {
	if err := validateChannelName(channelName); err != nil {
		return err
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	if _, exists := o.channels[channelName]; exists {
		return fmt.Errorf("channel %s already exists", channelName)
	}

	// 채널용 MSP 생성
	channelMSP := msp.NewFabricMSP()
	config := &msp.MSPConfig{
		Name: fmt.Sprintf("%s.%s", o.mspID, channelName),
		CryptoConfig: &msp.FabricCryptoConfig{
			SignatureHashFamily:            DefaultHashFamily,
			IdentityIdentifierHashFunction: DefaultHashFunction,
		},
	}

	if err := channelMSP.Setup(config); err != nil {
		return fmt.Errorf("failed to setup channel MSP: %w", err)
	}

	o.channels[channelName] = &Channel{
		Name:   channelName,
		Blocks: make([]*Block, 0),
		MSP:    channelMSP,
	}

	logger.Infof("Channel '%s' created successfully", channelName)
	return nil
}

// ValidateTransaction 트랜잭션 검증 (MSP 사용)
func (o *Orderer) ValidateTransaction(channelID string, serializedIdentity []byte, signature []byte, payload []byte) error {
	if channelID == "" {
		return fmt.Errorf("channel ID cannot be empty")
	}
	if len(serializedIdentity) == 0 {
		return fmt.Errorf("serialized identity cannot be empty")
	}
	if len(signature) == 0 {
		return fmt.Errorf("signature cannot be empty")
	}
	if len(payload) == 0 {
		return fmt.Errorf("payload cannot be empty")
	}

	o.mutex.RLock()
	channel, exists := o.channels[channelID]
	o.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("channel %s not found", channelID)
	}

	// Identity 역직렬화
	identity, err := channel.MSP.DeserializeIdentity(serializedIdentity)
	if err != nil {
		return fmt.Errorf("failed to deserialize identity: %w", err)
	}

	// Identity 검증
	if err := channel.MSP.ValidateIdentity(identity); err != nil {
		return fmt.Errorf("invalid identity: %w", err)
	}

	// 서명 검증
	if err := identity.Verify(payload, signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
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
func (o *Orderer) GetBlock(blockNumber uint64) (*Block, error) {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	if blockNumber >= uint64(len(o.blocks)) {
		return nil, fmt.Errorf("block %d not found", blockNumber)
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
func (o *Orderer) GetChannel(channelName string) (*Channel, error) {
	if channelName == "" {
		return nil, fmt.Errorf("channel name cannot be empty")
	}

	o.mutex.RLock()
	defer o.mutex.RUnlock()

	channel, exists := o.channels[channelName]
	if !exists {
		return nil, fmt.Errorf("channel %s not found", channelName)
	}

	return channel, nil
}

// BootstrapNetwork 네트워크 부트스트랩 (제네시스 블록 생성)
func (o *Orderer) BootstrapNetwork(genesisConfig *GenesisConfig) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if err := o.validateBootstrapPreconditions(genesisConfig); err != nil {
		return err
	}

	logger.Info("Starting network bootstrap process")

	// 1. 제네시스 블록 생성
	genesisBlock, err := o.generateGenesisBlock(genesisConfig)
	if err != nil {
		return fmt.Errorf("failed to generate genesis block: %w", err)
	}

	// 2. 네트워크 상태 초기화
	if err := o.initializeNetworkState(genesisBlock, genesisConfig); err != nil {
		return fmt.Errorf("failed to initialize network state: %w", err)
	}

	// 3. 부트스트랩 완료
	o.isBootstrapped = true
	o.logBootstrapSuccess(genesisConfig)

	return nil
}

// validateBootstrapPreconditions 부트스트랩 전제조건 검증
func (o *Orderer) validateBootstrapPreconditions(genesisConfig *GenesisConfig) error {
	if o.isBootstrapped {
		return fmt.Errorf("network is already bootstrapped")
	}

	if genesisConfig == nil {
		return fmt.Errorf("genesis config cannot be nil")
	}

	return nil
}

// generateGenesisBlock 제네시스 블록 생성
func (o *Orderer) generateGenesisBlock(genesisConfig *GenesisConfig) (*GenesisBlock, error) {
	generator, err := NewGenesisBlockGenerator(genesisConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create genesis block generator: %w", err)
	}

	genesisBlock, err := generator.GenerateGenesisBlock()
	if err != nil {
		return nil, fmt.Errorf("failed to generate genesis block: %w", err)
	}

	logger.Info("Genesis block generated successfully",
		"blockNumber", genesisBlock.Header.Number,
		"timestamp", time.Unix(genesisBlock.Header.Timestamp, 0).Format(time.RFC3339),
		"systemChannel", genesisConfig.SystemChannel.Name)

	return genesisBlock, nil
}

// initializeNetworkState 네트워크 상태 초기화
func (o *Orderer) initializeNetworkState(genesisBlock *GenesisBlock, genesisConfig *GenesisConfig) error {
	// 제네시스 블록을 첫 번째 블록으로 설정
	o.genesisBlock = genesisBlock
	o.systemChannel = genesisConfig.SystemChannel.Name

	// 제네시스 블록을 일반 블록 형태로 변환하여 저장
	block := o.convertGenesisBlockToBlock(genesisBlock)
	o.blocks = append(o.blocks, block)
	o.currentBlock = block

	// 시스템 채널 생성
	if err := o.createSystemChannel(genesisConfig); err != nil {
		return fmt.Errorf("failed to create system channel: %w", err)
	}

	return nil
}

// logBootstrapSuccess 부트스트랩 성공 로그 출력
func (o *Orderer) logBootstrapSuccess(genesisConfig *GenesisConfig) {
	logger.Info("Network bootstrap completed successfully",
		"networkName", genesisConfig.NetworkName,
		"consortium", genesisConfig.ConsortiumName,
		"ordererOrgs", len(genesisConfig.OrdererOrgs),
		"peerOrgs", len(genesisConfig.PeerOrgs))
}

// convertGenesisBlockToBlock 제네시스 블록을 일반 블록으로 변환
func (o *Orderer) convertGenesisBlockToBlock(genesisBlock *GenesisBlock) *Block {
	// 제네시스 블록 데이터를 JSON으로 직렬화
	data, err := json.Marshal(genesisBlock.Data)
	if err != nil {
		logger.Warnf("Failed to marshal genesis block data: %v", err)
		data = []byte("genesis_block_data")
	}

	return &Block{
		Number:       genesisBlock.Header.Number,
		PreviousHash: genesisBlock.Header.PreviousHash,
		Data:         data,
		Timestamp:    time.Unix(genesisBlock.Header.Timestamp, 0),
	}
}

// createSystemChannel 시스템 채널 생성
func (o *Orderer) createSystemChannel(genesisConfig *GenesisConfig) error {
	systemChannelName := genesisConfig.SystemChannel.Name

	// 시스템 채널용 MSP 생성
	systemMSP := msp.NewFabricMSP()
	config := &msp.MSPConfig{
		Name: fmt.Sprintf("%s.%s", o.mspID, systemChannelName),
		CryptoConfig: &msp.FabricCryptoConfig{
			SignatureHashFamily:            DefaultHashFamily,
			IdentityIdentifierHashFunction: DefaultHashFunction,
		},
		NodeOUs: &msp.FabricNodeOUs{
			Enable: true,
			OrdererOUIdentifier: &msp.FabricOUIdentifier{
				OrganizationalUnitIdentifier: DefaultOrdererOU,
			},
		},
	}

	if err := systemMSP.Setup(config); err != nil {
		return fmt.Errorf("failed to setup system channel MSP: %w", err)
	}

	// 시스템 채널 생성
	o.channels[systemChannelName] = &Channel{
		Name:   systemChannelName,
		Blocks: []*Block{o.currentBlock}, // 제네시스 블록 포함
		MSP:    systemMSP,
	}

	logger.Infof("System channel '%s' created successfully", systemChannelName)
	return nil
}

// IsBootstrapped 네트워크 부트스트랩 상태 확인
func (o *Orderer) IsBootstrapped() bool {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.isBootstrapped
}

// GetGenesisBlock 제네시스 블록 반환
func (o *Orderer) GetGenesisBlock() *GenesisBlock {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.genesisBlock
}

// GetSystemChannel 시스템 채널 이름 반환
func (o *Orderer) GetSystemChannel() string {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.systemChannel
}

// SaveGenesisBlock 제네시스 블록을 파일로 저장
func (o *Orderer) SaveGenesisBlock(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	o.mutex.RLock()
	genesisBlock := o.genesisBlock
	o.mutex.RUnlock()

	if genesisBlock == nil {
		return fmt.Errorf("no genesis block to save")
	}

	data, err := json.MarshalIndent(genesisBlock, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal genesis block: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write genesis block file: %w", err)
	}

	logger.Infof("Genesis block saved to %s", filePath)
	return nil
}

// LoadGenesisBlock 파일에서 제네시스 블록 로드
func (o *Orderer) LoadGenesisBlock(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read genesis block file: %w", err)
	}

	var genesisBlock GenesisBlock
	if err := json.Unmarshal(data, &genesisBlock); err != nil {
		return fmt.Errorf("failed to unmarshal genesis block: %w", err)
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.genesisBlock = &genesisBlock
	o.isBootstrapped = true

	// 제네시스 블록을 첫 번째 블록으로 설정
	if len(o.blocks) == 0 {
		block := o.convertGenesisBlockToBlock(&genesisBlock)
		o.blocks = append(o.blocks, block)
		o.currentBlock = block
	}

	logger.Infof("Genesis block loaded from %s", filePath)
	return nil
}

// validateChannelName validates channel name according to Fabric rules
func validateChannelName(channelName string) error {
	if channelName == "" {
		return fmt.Errorf("channel name cannot be empty")
	}

	if len(channelName) < MinChannelNameLength {
		return fmt.Errorf("channel name must be at least %d characters", MinChannelNameLength)
	}

	if len(channelName) > MaxChannelNameLength {
		return fmt.Errorf("channel name cannot exceed %d characters", MaxChannelNameLength)
	}

	// Channel names must be lowercase and contain only alphanumeric characters, dots, and dashes
	for _, char := range channelName {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '.' || char == '-') {
			return fmt.Errorf("channel name contains invalid character '%c'. Only lowercase letters, numbers, dots, and dashes are allowed", char)
		}
	}

	// Channel name cannot start or end with a dot or dash
	if channelName[0] == '.' || channelName[0] == '-' ||
		channelName[len(channelName)-1] == '.' || channelName[len(channelName)-1] == '-' {
		return fmt.Errorf("channel name cannot start or end with '.' or '-'")
	}

	return nil
}
