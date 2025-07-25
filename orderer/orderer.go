package orderer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ddr4869/minifab/common/common"
	"github.com/ddr4869/minifab/common/configtx"
	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
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
	channels       map[string]*common.Channel
	msp            msp.MSP
	mspID          string
	genesisBlock   *configtx.GenesisConfig
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
		blocks:   make([]*Block, 0),
		channels: make(map[string]*common.Channel),
		msp:      fabricMSP,
		mspID:    mspID,
	}, nil
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
func (o *Orderer) BootstrapNetwork(genesisConfig *configtx.GenesisConfig) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if err := o.validateBootstrapPreconditions(genesisConfig); err != nil {
		return err
	}

	logger.Info("Starting network bootstrap process")

	// 1. 제네시스 블록 생성
	err := o.generateGenesisBlock(genesisConfig)
	if err != nil {
		return errors.Wrap(err, "failed to generate genesis block")
	}

	// 2. 부트스트랩 완료
	o.isBootstrapped = true
	o.logBootstrapSuccess(genesisConfig)

	return nil
}

// validateBootstrapPreconditions 부트스트랩 전제조건 검증
func (o *Orderer) validateBootstrapPreconditions(genesisConfig *configtx.GenesisConfig) error {
	if o.isBootstrapped {
		return errors.New("network is already bootstrapped")
	}

	if genesisConfig == nil {
		return errors.New("genesis config cannot be nil")
	}

	return nil
}

// generateGenesisBlock 제네시스 블록 생성
func (o *Orderer) generateGenesisBlock(genesisConfig *configtx.GenesisConfig) error {

	// convert configtx to JSON file
	jsonData, err := json.Marshal(genesisConfig)
	if err != nil {
		return errors.Wrap(err, "failed to marshal genesis config")
	}

	// save json file
	os.WriteFile("genesis.json", jsonData, 0644)

	return nil
}

// logBootstrapSuccess 부트스트랩 성공 로그 출력
func (o *Orderer) logBootstrapSuccess(genesisConfig *configtx.GenesisConfig) {
	logger.Info("Network bootstrap completed successfully",
		"networkName", genesisConfig.NetworkName,
		"ordererOrgs", len(genesisConfig.OrdererOrgs))
}

// CreateGenesisConfigFromConfigTx configtx.yaml 파일에서 GenesisConfig 생성
func CreateGenesisConfigFromConfigTx(configTxPath string) (*configtx.GenesisConfig, error) {
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
	var configTx configtx.ConfigTx
	if err := yaml.Unmarshal(data, &configTx); err != nil {
		return nil, errors.Wrap(err, "failed to parse configtx YAML")
	}

	// ConfigTxYAML을 GenesisConfig로 변환
	genesisConfig, err := convertConfigTxToGenesisConfig(&configTx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert configtx to genesis config")
	}

	logger.Infof("Successfully loaded configuration from %s", configTxPath)
	logger.Infof("Network: %s", genesisConfig.NetworkName)
	logger.Infof("Orderer Organizations: %d", len(genesisConfig.OrdererOrgs))

	return genesisConfig, nil
}

// convertConfigTxToGenesisConfig ConfigTxYAML을 GenesisConfig로 변환
func convertConfigTxToGenesisConfig(configTx *configtx.ConfigTx) (*configtx.GenesisConfig, error) {
	// 기본값 설정
	networkName := configtx.DefaultNetworkName
	consortiumName := configtx.DefaultConsortiumName
	systemChannelName := configtx.DefaultSystemChannel

	// Organization 분류
	var ordererOrgs []*configtx.OrganizationConfig
	var peerOrgs []*configtx.OrganizationConfig

	// 현재 작업 디렉토리 가져오기 (프로젝트 루트)
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current working directory")
	}

	for _, org := range configTx.Organizations {
		orgConfig := &configtx.OrganizationConfig{
			Name:    org.Name,
			ID:      org.ID,
			MSPType: configtx.MSPTypeBCCSP,
			// Policies 필드 없음
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
		batchTimeout = configtx.DefaultBatchTimeout
	}

	// 정책: string 그대로 저장
	channelPolicies := map[string]*configtx.Policy{
		"Policy": {Type: "Simple", Rule: configTx.Channel.Policies},
	}

	// GenesisConfig 생성
	genesisConfig := &configtx.GenesisConfig{
		NetworkName:    networkName,
		ConsortiumName: consortiumName,
		OrdererOrgs:    ordererOrgs,
		PeerOrgs:       peerOrgs,
		SystemChannel: &configtx.SystemChannelConfig{
			Name:       systemChannelName,
			Consortium: consortiumName,
			Policies:   channelPolicies,
		},
		Policies:     channelPolicies,
		BatchSize:    batchSize,
		BatchTimeout: batchTimeout,
	}

	// 검증
	if err := configtx.ValidateGenesisConfig(genesisConfig); err != nil {
		return nil, errors.Wrap(err, "generated genesis config is invalid")
	}

	return genesisConfig, nil
}

// convertPoliciesFromYAML YAML 정책을 GenesisConfig 정책으로 변환
func convertPoliciesFromYAML(yamlPolicies map[string]configtx.Policy) map[string]*configtx.Policy {
	if yamlPolicies == nil {
		return make(map[string]*configtx.Policy)
	}

	policies := make(map[string]*configtx.Policy)
	for name, yamlPolicy := range yamlPolicies {
		policy := &configtx.Policy{
			Type: yamlPolicy.Type,
		}

		switch yamlPolicy.Type {
		case configtx.PolicyTypeImplicitMeta:
			// ImplicitMeta 정책 파싱 (예: "ANY Readers", "MAJORITY Admins")
			rule := parseImplicitMetaRule(yamlPolicy.Rule.(string))
			policy.Rule = rule
		case configtx.PolicyTypeSignature:
			// Signature 정책 파싱 (현재는 원본 규칙 문자열을 그대로 사용)
			policy.Rule = yamlPolicy.Rule
		default:
			// 기타 정책 타입
			policy.Rule = yamlPolicy.Rule
		}
		policies[name] = policy
	}

	return policies
}

// parseImplicitMetaRule ImplicitMeta 정책 규칙 파싱
func parseImplicitMetaRule(rule string) *configtx.ImplicitMetaRule {
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
		return &configtx.ImplicitMetaRule{
			Rule:      parts[0],
			SubPolicy: parts[1],
		}
	}

	// 파싱 실패 시 기본값
	return &configtx.ImplicitMetaRule{
		Rule:      configtx.PolicyRuleAny,
		SubPolicy: "Readers",
	}
}

// convertBatchSizeFromYAML YAML BatchSize를 GenesisConfig BatchSize로 변환
func convertBatchSizeFromYAML(yamlBatchSize configtx.BatchSize) (*configtx.BatchSizeConfig, error) {
	absoluteMaxBytes, err := parseBatchSizeBytes(yamlBatchSize.AbsoluteMaxBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse absolute max bytes")
	}

	preferredMaxBytes, err := parseBatchSizeBytes(yamlBatchSize.PreferredMaxBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse preferred max bytes")
	}

	return &configtx.BatchSizeConfig{
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
