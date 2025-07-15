package orderer

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
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
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
		return nil, errors.New("block data cannot be empty")
	}

	if len(data) > MaxBlockDataSize {
		return nil, errors.Errorf("block data size %d exceeds maximum allowed size %d", len(data), MaxBlockDataSize)
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
		return errors.Errorf("channel %s already exists", channelName)
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
		return errors.Wrap(err, "failed to setup channel MSP")
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
		return errors.New("channel ID cannot be empty")
	}
	if len(serializedIdentity) == 0 {
		return errors.New("serialized identity cannot be empty")
	}
	if len(signature) == 0 {
		return errors.New("signature cannot be empty")
	}
	if len(payload) == 0 {
		return errors.New("payload cannot be empty")
	}

	o.mutex.RLock()
	channel, exists := o.channels[channelID]
	o.mutex.RUnlock()

	if !exists {
		return errors.Errorf("channel %s not found", channelID)
	}

	// Identity 역직렬화
	identity, err := channel.MSP.DeserializeIdentity(serializedIdentity)
	if err != nil {
		return errors.Wrap(err, "failed to deserialize identity")
	}

	// Identity 검증
	if err := channel.MSP.ValidateIdentity(identity); err != nil {
		return errors.Wrap(err, "invalid identity")
	}

	// 서명 검증
	if err := identity.Verify(payload, signature); err != nil {
		return errors.Wrap(err, "signature verification failed")
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
func (o *Orderer) GetChannel(channelName string) (*Channel, error) {
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
		return errors.Wrap(err, "failed to generate genesis block")
	}

	// 2. 네트워크 상태 초기화
	if err := o.initializeNetworkState(genesisBlock, genesisConfig); err != nil {
		return errors.Wrap(err, "failed to initialize network state")
	}

	// 3. 부트스트랩 완료
	o.isBootstrapped = true
	o.logBootstrapSuccess(genesisConfig)

	return nil
}

// validateBootstrapPreconditions 부트스트랩 전제조건 검증
func (o *Orderer) validateBootstrapPreconditions(genesisConfig *GenesisConfig) error {
	if o.isBootstrapped {
		return errors.New("network is already bootstrapped")
	}

	if genesisConfig == nil {
		return errors.New("genesis config cannot be nil")
	}

	return nil
}

// generateGenesisBlock 제네시스 블록 생성
func (o *Orderer) generateGenesisBlock(genesisConfig *GenesisConfig) (*GenesisBlock, error) {
	generator, err := NewGenesisBlockGenerator(genesisConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create genesis block generator")
	}

	genesisBlock, err := generator.GenerateGenesisBlock()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate genesis block")
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
		return errors.Wrap(err, "failed to create system channel")
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
		return errors.Wrap(err, "failed to setup system channel MSP")
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
		return errors.New("file path cannot be empty")
	}

	o.mutex.RLock()
	genesisBlock := o.genesisBlock
	o.mutex.RUnlock()

	if genesisBlock == nil {
		return errors.New("no genesis block to save")
	}

	data, err := json.MarshalIndent(genesisBlock, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal genesis block")
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return errors.Wrap(err, "failed to write genesis block file")
	}

	logger.Infof("Genesis block saved to %s", filePath)
	return nil
}

// LoadGenesisBlock 파일에서 제네시스 블록 로드
func (o *Orderer) LoadGenesisBlock(filePath string) error {
	if filePath == "" {
		return errors.New("file path cannot be empty")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return errors.Wrap(err, "failed to read genesis block file")
	}

	var genesisBlock GenesisBlock
	if err := json.Unmarshal(data, &genesisBlock); err != nil {
		return errors.Wrap(err, "failed to unmarshal genesis block")
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
		return errors.New("channel name cannot be empty")
	}

	if len(channelName) < MinChannelNameLength {
		return errors.Errorf("channel name must be at least %d characters", MinChannelNameLength)
	}

	if len(channelName) > MaxChannelNameLength {
		return errors.Errorf("channel name cannot exceed %d characters", MaxChannelNameLength)
	}

	// Channel names must be lowercase and contain only alphanumeric characters, dots, and dashes
	for _, char := range channelName {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '.' || char == '-') {
			return errors.Errorf("channel name contains invalid character '%c'. Only lowercase letters, numbers, dots, and dashes are allowed", char)
		}
	}

	// Channel name cannot start or end with a dot or dash
	if channelName[0] == '.' || channelName[0] == '-' ||
		channelName[len(channelName)-1] == '.' || channelName[len(channelName)-1] == '-' {
		return errors.Errorf("channel name cannot start or end with '.' or '-'")
	}

	return nil
}

// CreateGenesisConfigFromConfigTx configtx.yaml 파일에서 GenesisConfig 생성
func CreateGenesisConfigFromConfigTx(configTxPath string) (*GenesisConfig, error) {
	if configTxPath == "" {
		return nil, errors.Errorf("configtx path cannot be empty")
	}

	// configtx.yaml 파일 존재 확인
	if _, err := os.Stat(configTxPath); os.IsNotExist(err) {
		return nil, errors.Errorf("configtx file does not exist: %s", configTxPath)
	}

	// configtx.yaml 파일 읽기
	data, err := os.ReadFile(configTxPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read configtx file")
	}

	// YAML 파싱
	var configTx ConfigTxYAML
	if err := yaml.Unmarshal(data, &configTx); err != nil {
		return nil, errors.Wrap(err, "failed to parse configtx YAML")
	}

	// ConfigTxYAML을 GenesisConfig로 변환
	genesisConfig, err := convertConfigTxToGenesisConfig(&configTx, configTxPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert configtx to genesis config")
	}

	logger.Infof("Successfully loaded configuration from %s", configTxPath)
	logger.Infof("Network: %s, Consortium: %s", genesisConfig.NetworkName, genesisConfig.ConsortiumName)
	logger.Infof("Orderer Organizations: %d, Peer Organizations: %d",
		len(genesisConfig.OrdererOrgs), len(genesisConfig.PeerOrgs))

	return genesisConfig, nil
}

// ConfigTxYAML configtx.yaml 파일 구조체 정의
type ConfigTxYAML struct {
	Organizations []OrganizationYAML `yaml:"Organizations"`
	Application   ApplicationYAML    `yaml:"Application"`
	Orderer       OrdererYAML        `yaml:"Orderer"`
	Channel       ChannelYAML        `yaml:"Channel"`
}

// OrganizationYAML YAML의 Organization 구조체
type OrganizationYAML struct {
	Name             string                `yaml:"Name"`
	ID               string                `yaml:"ID"`
	MSPDir           string                `yaml:"MSPDir"`
	Policies         map[string]PolicyYAML `yaml:"Policies"`
	OrdererEndpoints []string              `yaml:"OrdererEndpoints,omitempty"`
	AnchorPeers      []AnchorPeerYAML      `yaml:"AnchorPeers,omitempty"`
}

// AnchorPeerYAML YAML의 AnchorPeer 구조체
type AnchorPeerYAML struct {
	Host string `yaml:"Host"`
	Port int    `yaml:"Port"`
}

// PolicyYAML YAML의 Policy 구조체
type PolicyYAML struct {
	Type string `yaml:"Type"`
	Rule string `yaml:"Rule"`
}

// ApplicationYAML YAML의 Application 구조체
type ApplicationYAML struct {
	Organizations []interface{}         `yaml:"Organizations"`
	Policies      map[string]PolicyYAML `yaml:"Policies"`
}

// OrdererYAML YAML의 Orderer 구조체
type OrdererYAML struct {
	OrdererType   string                `yaml:"OrdererType"`
	BatchTimeout  string                `yaml:"BatchTimeout"`
	BatchSize     BatchSizeYAML         `yaml:"BatchSize"`
	Organizations []interface{}         `yaml:"Organizations"`
	Policies      map[string]PolicyYAML `yaml:"Policies"`
}

// BatchSizeYAML YAML의 BatchSize 구조체
type BatchSizeYAML struct {
	MaxMessageCount   int    `yaml:"MaxMessageCount"`
	AbsoluteMaxBytes  string `yaml:"AbsoluteMaxBytes"`
	PreferredMaxBytes string `yaml:"PreferredMaxBytes"`
}

// ChannelYAML YAML의 Channel 구조체
type ChannelYAML struct {
	Policies map[string]PolicyYAML `yaml:"Policies"`
}

// convertConfigTxToGenesisConfig ConfigTxYAML을 GenesisConfig로 변환
func convertConfigTxToGenesisConfig(configTx *ConfigTxYAML, configTxPath string) (*GenesisConfig, error) {
	// 기본값 설정
	networkName := DefaultNetworkName
	consortiumName := DefaultConsortiumName
	systemChannelName := DefaultSystemChannel

	// Organization 분류
	var ordererOrgs []*OrganizationConfig
	var peerOrgs []*OrganizationConfig

	// 현재 작업 디렉토리 가져오기 (프로젝트 루트)
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current working directory")
	}

	for _, org := range configTx.Organizations {
		orgConfig := &OrganizationConfig{
			Name:     org.Name,
			ID:       org.ID,
			MSPType:  MSPTypeBCCSP,
			Policies: convertPoliciesFromYAML(org.Policies),
		}

		// MSPDir 경로 처리 - 상대 경로는 프로젝트 루트를 기준으로 함
		if filepath.IsAbs(org.MSPDir) {
			orgConfig.MSPDir = org.MSPDir
		} else {
			// 상대 경로는 프로젝트 루트(workingDir)를 기준으로 함
			orgConfig.MSPDir = filepath.Join(workingDir, org.MSPDir)
		}

		// OrdererEndpoints가 있으면 orderer 조직
		if len(org.OrdererEndpoints) > 0 {
			ordererOrgs = append(ordererOrgs, orgConfig)
		}
		// AnchorPeers가 있으면 peer 조직
		if len(org.AnchorPeers) > 0 {
			peerOrgs = append(peerOrgs, orgConfig)
		}
	}

	// BatchSize 변환
	batchSize, err := convertBatchSizeFromYAML(configTx.Orderer.BatchSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert batch size")
	}

	// BatchTimeout 처리
	batchTimeout := configTx.Orderer.BatchTimeout
	if batchTimeout == "" {
		batchTimeout = DefaultBatchTimeout
	}

	// GenesisConfig 생성
	genesisConfig := &GenesisConfig{
		NetworkName:    networkName,
		ConsortiumName: consortiumName,
		OrdererOrgs:    ordererOrgs,
		PeerOrgs:       peerOrgs,
		SystemChannel: &SystemChannelConfig{
			Name:         systemChannelName,
			Consortium:   consortiumName,
			Capabilities: map[string]bool{CapabilityV2_0: true},
			Policies:     convertPoliciesFromYAML(configTx.Channel.Policies),
		},
		Capabilities: map[string]bool{CapabilityV2_0: true},
		Policies:     convertPoliciesFromYAML(configTx.Channel.Policies),
		BatchSize:    batchSize,
		BatchTimeout: batchTimeout,
	}

	// 검증
	if err := validateGenesisConfig(genesisConfig); err != nil {
		return nil, errors.Wrap(err, "generated genesis config is invalid")
	}

	return genesisConfig, nil
}

// convertPoliciesFromYAML YAML 정책을 GenesisConfig 정책으로 변환
func convertPoliciesFromYAML(yamlPolicies map[string]PolicyYAML) map[string]*Policy {
	if yamlPolicies == nil {
		return make(map[string]*Policy)
	}

	policies := make(map[string]*Policy)
	for name, yamlPolicy := range yamlPolicies {
		policy := &Policy{
			Type: yamlPolicy.Type,
		}

		// 정책 규칙 변환
		if yamlPolicy.Type == PolicyTypeImplicitMeta {
			// ImplicitMeta 정책 파싱 (예: "ANY Readers", "MAJORITY Admins")
			rule := parseImplicitMetaRule(yamlPolicy.Rule)
			policy.Rule = rule
		} else if yamlPolicy.Type == PolicyTypeSignature {
			// Signature 정책 파싱 (현재는 원본 규칙 문자열을 그대로 사용)
			policy.Rule = yamlPolicy.Rule
		} else {
			// 기타 정책 타입
			policy.Rule = yamlPolicy.Rule
		}

		policies[name] = policy
	}

	return policies
}

// parseImplicitMetaRule ImplicitMeta 정책 규칙 파싱
func parseImplicitMetaRule(rule string) *ImplicitMetaRule {
	// "ANY Readers", "MAJORITY Admins" 등의 형태를 파싱
	parts := make([]string, 0, 2)
	for _, part := range []string{"ANY", "MAJORITY", "ALL"} {
		if len(rule) > len(part) && rule[:len(part)] == part {
			parts = append(parts, part)
			if len(rule) > len(part)+1 {
				parts = append(parts, rule[len(part)+1:])
			}
			break
		}
	}

	if len(parts) >= 2 {
		return &ImplicitMetaRule{
			Rule:      parts[0],
			SubPolicy: parts[1],
		}
	}

	// 파싱 실패 시 기본값
	return &ImplicitMetaRule{
		Rule:      PolicyRuleAny,
		SubPolicy: "Readers",
	}
}

// convertBatchSizeFromYAML YAML BatchSize를 GenesisConfig BatchSize로 변환
func convertBatchSizeFromYAML(yamlBatchSize BatchSizeYAML) (*BatchSizeConfig, error) {
	absoluteMaxBytes, err := parseBatchSizeBytes(yamlBatchSize.AbsoluteMaxBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse absolute max bytes")
	}

	preferredMaxBytes, err := parseBatchSizeBytes(yamlBatchSize.PreferredMaxBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse preferred max bytes")
	}

	return &BatchSizeConfig{
		MaxMessageCount:   uint32(yamlBatchSize.MaxMessageCount),
		AbsoluteMaxBytes:  absoluteMaxBytes,
		PreferredMaxBytes: preferredMaxBytes,
	}, nil
}

// parseBatchSizeBytes 크기 문자열을 바이트 수로 변환 ("128 MB" -> 134217728)
func parseBatchSizeBytes(sizeStr string) (uint32, error) {
	if sizeStr == "" {
		return 0, errors.New("size string cannot be empty")
	}

	var value uint32
	var unit string

	// 숫자와 단위 분리
	n, err := fmt.Sscanf(sizeStr, "%d %s", &value, &unit)
	if err != nil || n != 2 {
		// 단위 없이 숫자만 있는 경우
		if n, err := fmt.Sscanf(sizeStr, "%d", &value); err != nil || n != 1 {
			return 0, errors.Errorf("failed to parse size: %s", sizeStr)
		}
		return value, nil
	}

	// 단위에 따른 배수 적용
	switch unit {
	case "KB":
		return value * 1024, nil
	case "MB":
		return value * 1024 * 1024, nil
	case "GB":
		return value * 1024 * 1024 * 1024, nil
	default:
		return 0, errors.Errorf("unsupported size unit: %s", unit)
	}
}
