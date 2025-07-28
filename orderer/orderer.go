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
	// convert configtx to JSON file
	jsonData, err := json.Marshal(genesisConfig)
	if err != nil {
		return errors.Wrap(err, "failed to marshal genesis config")
	}

	// save json file
	os.WriteFile("genesis.json", jsonData, 0644)

	return nil
}
