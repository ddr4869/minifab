package storage

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/types"
	"github.com/pkg/errors"
)

// BlockStorage handles persistent storage of blocks with atomic writes
type BlockStorage struct {
	mutex       sync.RWMutex
	storagePath string
	// In-memory cache for quick access
	channelHeights  map[string]uint64
	lastBlockHash   map[string][]byte
	committedBlocks map[string]map[uint64]bool
}

// StoredBlock represents a block stored on disk
type StoredBlock struct {
	Block       *types.Block `json:"block"`
	ChannelID   string       `json:"channel_id"`
	StoredAt    time.Time    `json:"stored_at"`
	IsCommitted bool         `json:"is_committed"`
	BlockHash   []byte       `json:"block_hash"`
}

// NewBlockStorage creates a new block storage instance
func NewBlockStorage() *BlockStorage {
	storagePath := "./blocks" // Default storage path

	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		logger.Errorf("Failed to create storage directory: %v", err)
	}

	storage := &BlockStorage{
		storagePath:     storagePath,
		channelHeights:  make(map[string]uint64),
		lastBlockHash:   make(map[string][]byte),
		committedBlocks: make(map[string]map[uint64]bool),
	}

	// Initialize storage by loading existing blocks
	if err := storage.initialize(); err != nil {
		logger.Errorf("Failed to initialize block storage: %v", err)
	}

	return storage
}

// initialize loads existing blocks from disk to rebuild in-memory state
func (bs *BlockStorage) initialize() error {
	logger.Info("Initializing block storage...")

	// Walk through storage directory to find existing blocks
	return filepath.Walk(bs.storagePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-JSON files
		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		// Load block from file
		storedBlock, err := bs.loadBlockFromFile(path)
		if err != nil {
			logger.Warnf("Failed to load block from %s: %v", path, err)
			return nil // Continue with other files
		}

		// Update in-memory state
		channelID := storedBlock.ChannelID
		blockNumber := storedBlock.Block.Number

		// Update channel height
		if currentHeight, exists := bs.channelHeights[channelID]; !exists || blockNumber >= currentHeight {
			bs.channelHeights[channelID] = blockNumber + 1
			bs.lastBlockHash[channelID] = storedBlock.BlockHash
		}

		// Track committed blocks
		if bs.committedBlocks[channelID] == nil {
			bs.committedBlocks[channelID] = make(map[uint64]bool)
		}
		bs.committedBlocks[channelID][blockNumber] = storedBlock.IsCommitted

		return nil
	})
}

// StoreBlock stores a block atomically to disk
func (bs *BlockStorage) StoreBlock(channelID string, block *types.Block) error {
	if channelID == "" {
		return errors.New("channel ID cannot be empty")
	}
	if block == nil {
		return errors.New("block cannot be nil")
	}

	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	logger.Debugf("Storing block %d for channel %s", block.Number, channelID)

	// Calculate block hash
	blockHash := bs.calculateBlockHash(block)

	// Create stored block structure
	storedBlock := &StoredBlock{
		Block:       block,
		ChannelID:   channelID,
		StoredAt:    time.Now(),
		IsCommitted: false,
		BlockHash:   blockHash,
	}

	// Create channel directory if it doesn't exist
	channelDir := filepath.Join(bs.storagePath, channelID)
	if err := os.MkdirAll(channelDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create channel directory")
	}

	// Generate file path
	fileName := fmt.Sprintf("block_%d.json", block.Number)
	filePath := filepath.Join(channelDir, fileName)
	tempPath := filePath + ".tmp"

	// Serialize block to JSON
	data, err := json.MarshalIndent(storedBlock, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal block")
	}

	// Write to temporary file first (atomic write)
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return errors.Wrap(err, "failed to write temporary block file")
	}

	// Atomically rename temporary file to final file
	if err := os.Rename(tempPath, filePath); err != nil {
		// Clean up temporary file on failure
		os.Remove(tempPath)
		return errors.Wrap(err, "failed to rename temporary block file")
	}

	// Update in-memory state
	bs.channelHeights[channelID] = block.Number + 1
	bs.lastBlockHash[channelID] = blockHash

	if bs.committedBlocks[channelID] == nil {
		bs.committedBlocks[channelID] = make(map[uint64]bool)
	}
	bs.committedBlocks[channelID][block.Number] = false

	logger.Debugf("Successfully stored block %d for channel %s", block.Number, channelID)
	return nil
}

// GetBlock retrieves a specific block from storage
func (bs *BlockStorage) GetBlock(channelID string, blockNumber uint64) (*types.Block, error) {
	if channelID == "" {
		return nil, errors.New("channel ID cannot be empty")
	}

	bs.mutex.RLock()
	defer bs.mutex.RUnlock()

	fileName := fmt.Sprintf("block_%d.json", blockNumber)
	filePath := filepath.Join(bs.storagePath, channelID, fileName)

	storedBlock, err := bs.loadBlockFromFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load block %d", blockNumber)
	}

	return storedBlock.Block, nil
}

// GetBlockRange retrieves a range of blocks from storage
func (bs *BlockStorage) GetBlockRange(channelID string, startBlock, endBlock uint64) ([]*types.Block, error) {
	if channelID == "" {
		return nil, errors.New("channel ID cannot be empty")
	}

	bs.mutex.RLock()
	defer bs.mutex.RUnlock()

	var blocks []*types.Block

	// Determine actual end block
	channelHeight := bs.channelHeights[channelID]
	if endBlock == 0 || endBlock > channelHeight {
		endBlock = channelHeight
	}

	// Load blocks in range
	for blockNum := startBlock; blockNum < endBlock; blockNum++ {
		block, err := bs.getBlockUnsafe(channelID, blockNum)
		if err != nil {
			logger.Warnf("Failed to load block %d: %v", blockNum, err)
			continue
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// getBlockUnsafe retrieves a block without locking (internal use)
func (bs *BlockStorage) getBlockUnsafe(channelID string, blockNumber uint64) (*types.Block, error) {
	fileName := fmt.Sprintf("block_%d.json", blockNumber)
	filePath := filepath.Join(bs.storagePath, channelID, fileName)

	storedBlock, err := bs.loadBlockFromFile(filePath)
	if err != nil {
		return nil, err
	}

	return storedBlock.Block, nil
}

// GetChannelHeight returns the current height of a channel
func (bs *BlockStorage) GetChannelHeight(channelID string) uint64 {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()

	return bs.channelHeights[channelID]
}

// GetLastBlockHash returns the hash of the last block in a channel
func (bs *BlockStorage) GetLastBlockHash(channelID string) []byte {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()

	return bs.lastBlockHash[channelID]
}

// MarkBlockCommitted marks a block as committed
func (bs *BlockStorage) MarkBlockCommitted(channelID string, blockNumber uint64) error {
	if channelID == "" {
		return errors.New("channel ID cannot be empty")
	}

	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	// Update in-memory state
	if bs.committedBlocks[channelID] == nil {
		bs.committedBlocks[channelID] = make(map[uint64]bool)
	}
	bs.committedBlocks[channelID][blockNumber] = true

	// Update file on disk
	fileName := fmt.Sprintf("block_%d.json", blockNumber)
	filePath := filepath.Join(bs.storagePath, channelID, fileName)

	storedBlock, err := bs.loadBlockFromFile(filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to load block %d for commit update", blockNumber)
	}

	storedBlock.IsCommitted = true

	// Write updated block back to disk
	data, err := json.MarshalIndent(storedBlock, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal updated block")
	}

	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return errors.Wrap(err, "failed to write updated block file")
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return errors.Wrap(err, "failed to update block file")
	}

	logger.Debugf("Marked block %d as committed for channel %s", blockNumber, channelID)
	return nil
}

// IsBlockCommitted checks if a block is committed
func (bs *BlockStorage) IsBlockCommitted(channelID string, blockNumber uint64) bool {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()

	if channelBlocks, exists := bs.committedBlocks[channelID]; exists {
		return channelBlocks[blockNumber]
	}
	return false
}

// loadBlockFromFile loads a stored block from a file
func (bs *BlockStorage) loadBlockFromFile(filePath string) (*StoredBlock, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read block file")
	}

	var storedBlock StoredBlock
	if err := json.Unmarshal(data, &storedBlock); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal block")
	}

	return &storedBlock, nil
}

// calculateBlockHash calculates the hash of a block
func (bs *BlockStorage) calculateBlockHash(block *types.Block) []byte {
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

// GetStorageStats returns storage statistics
func (bs *BlockStorage) GetStorageStats() map[string]any {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()

	stats := make(map[string]any)
	stats["storage_path"] = bs.storagePath
	stats["channels"] = len(bs.channelHeights)

	channelStats := make(map[string]any)
	for channelID, height := range bs.channelHeights {
		committedCount := 0
		if channelBlocks, exists := bs.committedBlocks[channelID]; exists {
			for _, committed := range channelBlocks {
				if committed {
					committedCount++
				}
			}
		}

		channelStats[channelID] = map[string]any{
			"height":           height,
			"committed_blocks": committedCount,
			"total_blocks":     height,
		}
	}
	stats["channel_stats"] = channelStats

	return stats
}

// Cleanup removes old block files (for maintenance)
func (bs *BlockStorage) Cleanup(channelID string, keepBlocks uint64) error {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	channelHeight := bs.channelHeights[channelID]
	if channelHeight <= keepBlocks {
		return nil // Nothing to cleanup
	}

	channelDir := filepath.Join(bs.storagePath, channelID)

	// Remove old block files
	for blockNum := uint64(0); blockNum < channelHeight-keepBlocks; blockNum++ {
		fileName := fmt.Sprintf("block_%d.json", blockNum)
		filePath := filepath.Join(channelDir, fileName)

		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			logger.Warnf("Failed to remove old block file %s: %v", filePath, err)
		}
	}

	logger.Infof("Cleaned up old blocks for channel %s, kept last %d blocks", channelID, keepBlocks)
	return nil
}
