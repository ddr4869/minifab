package peer

import (
	"context"
	"sync"
	"time"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/proto"
	"github.com/pkg/errors"
)

// BlockSynchronizer handles block synchronization for peer startup and recovery
type BlockSynchronizer struct {
	peer          *Peer
	ordererClient *OrdererClient
	blockStorage  *BlockStorage
	mutex         sync.RWMutex
	isRunning     bool
	stopChan      chan struct{}
}

// SyncConfig contains configuration for block synchronization
type SyncConfig struct {
	BatchSize       uint64        // Number of blocks to sync in each batch
	SyncInterval    time.Duration // Interval between sync attempts
	MaxRetries      int           // Maximum number of retry attempts
	RetryDelay      time.Duration // Delay between retries
	EnableStreaming bool          // Whether to use streaming for real-time sync
}

// DefaultSyncConfig returns default synchronization configuration
func DefaultSyncConfig() *SyncConfig {
	return &SyncConfig{
		BatchSize:       50,
		SyncInterval:    30 * time.Second,
		MaxRetries:      3,
		RetryDelay:      5 * time.Second,
		EnableStreaming: true,
	}
}

// NewBlockSynchronizer creates a new block synchronizer
func NewBlockSynchronizer(peer *Peer, ordererClient *OrdererClient, blockStorage *BlockStorage) *BlockSynchronizer {
	return &BlockSynchronizer{
		peer:          peer,
		ordererClient: ordererClient,
		blockStorage:  blockStorage,
		stopChan:      make(chan struct{}),
	}
}

// StartSync starts the block synchronization process
func (bs *BlockSynchronizer) StartSync(ctx context.Context, config *SyncConfig) error {
	bs.mutex.Lock()
	if bs.isRunning {
		bs.mutex.Unlock()
		return errors.New("synchronization is already running")
	}
	bs.isRunning = true
	bs.mutex.Unlock()

	logger.Info("Starting block synchronization service")

	// Start synchronization goroutine
	go bs.syncLoop(ctx, config)

	return nil
}

// StopSync stops the block synchronization process
func (bs *BlockSynchronizer) StopSync() {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	if !bs.isRunning {
		return
	}

	logger.Info("Stopping block synchronization service")
	close(bs.stopChan)
	bs.isRunning = false
}

// syncLoop is the main synchronization loop
func (bs *BlockSynchronizer) syncLoop(ctx context.Context, config *SyncConfig) {
	ticker := time.NewTicker(config.SyncInterval)
	defer ticker.Stop()

	// Perform initial sync for all channels
	if err := bs.performInitialSync(ctx, config); err != nil {
		logger.Errorf("Initial sync failed: %v", err)
	}

	// Start streaming sync if enabled
	if config.EnableStreaming {
		go bs.startStreamingSync(ctx, config)
	}

	// Periodic sync loop
	for {
		select {
		case <-ctx.Done():
			logger.Info("Block synchronization stopped due to context cancellation")
			return
		case <-bs.stopChan:
			logger.Info("Block synchronization stopped")
			return
		case <-ticker.C:
			if err := bs.performPeriodicSync(ctx, config); err != nil {
				logger.Errorf("Periodic sync failed: %v", err)
			}
		}
	}
}

// performInitialSync performs initial synchronization for all channels
func (bs *BlockSynchronizer) performInitialSync(ctx context.Context, config *SyncConfig) error {
	logger.Info("Performing initial block synchronization")

	// Get all channels that the peer has joined
	channels := bs.peer.channelManager.GetChannelNames()

	for _, channelID := range channels {
		if err := bs.syncChannel(ctx, channelID, config); err != nil {
			logger.Errorf("Failed to sync channel %s: %v", channelID, err)
			continue
		}
		logger.Infof("Successfully synced channel %s", channelID)
	}

	return nil
}

// performPeriodicSync performs periodic synchronization check
func (bs *BlockSynchronizer) performPeriodicSync(ctx context.Context, config *SyncConfig) error {
	logger.Debug("Performing periodic block synchronization check")

	channels := bs.peer.channelManager.GetChannelNames()

	for _, channelID := range channels {
		// Check if channel needs synchronization
		needsSync, err := bs.checkChannelNeedsSync(channelID)
		if err != nil {
			logger.Errorf("Failed to check sync status for channel %s: %v", channelID, err)
			continue
		}

		if needsSync {
			logger.Infof("Channel %s needs synchronization", channelID)
			if err := bs.syncChannel(ctx, channelID, config); err != nil {
				logger.Errorf("Failed to sync channel %s: %v", channelID, err)
			}
		}
	}

	return nil
}

// syncChannel synchronizes blocks for a specific channel
func (bs *BlockSynchronizer) syncChannel(ctx context.Context, channelID string, config *SyncConfig) error {
	// Get current local height
	localHeight := bs.blockStorage.GetChannelHeight(channelID)

	// Get remote height from orderer (we'll use a simple approach for now)
	// In a real implementation, we'd query the orderer for channel info
	remoteHeight := localHeight + 10 // Placeholder - assume there might be new blocks

	if localHeight >= remoteHeight {
		logger.Debugf("Channel %s is up to date (local: %d, remote: %d)", channelID, localHeight, remoteHeight)
		return nil
	}

	logger.Infof("Syncing channel %s from block %d to %d", channelID, localHeight, remoteHeight)

	// Sync blocks in batches
	for startBlock := localHeight; startBlock < remoteHeight; startBlock += config.BatchSize {
		endBlock := startBlock + config.BatchSize
		if endBlock > remoteHeight {
			endBlock = remoteHeight
		}

		if err := bs.syncBlockRange(ctx, channelID, startBlock, endBlock, config); err != nil {
			return errors.Wrapf(err, "failed to sync block range %d-%d", startBlock, endBlock)
		}
	}

	return nil
}

// syncBlockRange synchronizes a range of blocks for a channel
func (bs *BlockSynchronizer) syncBlockRange(ctx context.Context, channelID string, startBlock, endBlock uint64, config *SyncConfig) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		if attempt > 0 {
			logger.Infof("Retrying block sync for channel %s, attempt %d/%d", channelID, attempt+1, config.MaxRetries)
			time.Sleep(config.RetryDelay)
		}

		// Get blocks from orderer
		blocks, err := bs.ordererClient.GetBlockRange(channelID, startBlock, endBlock)
		if err != nil {
			lastErr = err
			logger.Warnf("Failed to get blocks from orderer (attempt %d): %v", attempt+1, err)
			continue
		}

		// Process and store each block
		for _, protoBlock := range blocks {
			// Convert to internal block format
			internalBlock := &Block{
				Number:       protoBlock.Number,
				PreviousHash: protoBlock.PreviousHash,
				Data:         protoBlock.DataHash,
				Timestamp:    time.Unix(protoBlock.Timestamp, 0),
			}

			// Store block
			if err := bs.blockStorage.StoreBlock(channelID, internalBlock); err != nil {
				lastErr = err
				logger.Errorf("Failed to store block %d: %v", protoBlock.Number, err)
				break
			}

			// Update channel with transactions
			if err := bs.updateChannelWithBlock(channelID, protoBlock); err != nil {
				logger.Warnf("Failed to update channel with block %d: %v", protoBlock.Number, err)
			}

			logger.Debugf("Synced block %d for channel %s", protoBlock.Number, channelID)
		}

		// If we got here without error, sync was successful
		return nil
	}

	return errors.Wrapf(lastErr, "failed to sync blocks after %d attempts", config.MaxRetries)
}

// startStreamingSync starts real-time block streaming
func (bs *BlockSynchronizer) startStreamingSync(ctx context.Context, config *SyncConfig) {
	logger.Info("Starting streaming block synchronization")

	channels := bs.peer.channelManager.GetChannelNames()

	for _, channelID := range channels {
		go bs.streamChannelBlocks(ctx, channelID, config)
	}
}

// streamChannelBlocks streams blocks for a specific channel
func (bs *BlockSynchronizer) streamChannelBlocks(ctx context.Context, channelID string, config *SyncConfig) {
	startBlock := bs.blockStorage.GetChannelHeight(channelID)

	logger.Infof("Starting block stream for channel %s from block %d", channelID, startBlock)

	blockChan, errorChan := bs.ordererClient.StreamBlocks(channelID, startBlock)

	for {
		select {
		case <-ctx.Done():
			return
		case <-bs.stopChan:
			return
		case block, ok := <-blockChan:
			if !ok {
				logger.Infof("Block stream closed for channel %s", channelID)
				return
			}

			// Process received block
			if err := bs.processStreamedBlock(channelID, block); err != nil {
				logger.Errorf("Failed to process streamed block %d for channel %s: %v",
					block.Number, channelID, err)
			}

		case err, ok := <-errorChan:
			if !ok {
				return
			}
			logger.Errorf("Block stream error for channel %s: %v", channelID, err)

			// Restart streaming after delay
			time.Sleep(config.RetryDelay)
			go bs.streamChannelBlocks(ctx, channelID, config)
			return
		}
	}
}

// processStreamedBlock processes a block received from streaming
func (bs *BlockSynchronizer) processStreamedBlock(channelID string, protoBlock *proto.Block) error {
	// Convert to internal block format
	internalBlock := &Block{
		Number:       protoBlock.Number,
		PreviousHash: protoBlock.PreviousHash,
		Data:         protoBlock.DataHash,
		Timestamp:    time.Unix(protoBlock.Timestamp, 0),
	}

	// Store block
	if err := bs.blockStorage.StoreBlock(channelID, internalBlock); err != nil {
		return errors.Wrap(err, "failed to store streamed block")
	}

	// Update channel with transactions
	if err := bs.updateChannelWithBlock(channelID, protoBlock); err != nil {
		logger.Warnf("Failed to update channel with streamed block %d: %v", protoBlock.Number, err)
	}

	logger.Infof("Processed streamed block %d for channel %s", protoBlock.Number, channelID)
	return nil
}

// updateChannelWithBlock updates channel state with block transactions
func (bs *BlockSynchronizer) updateChannelWithBlock(channelID string, protoBlock *proto.Block) error {
	channel, err := bs.peer.channelManager.GetChannel(channelID)
	if err != nil {
		return errors.Wrap(err, "failed to get channel")
	}

	// Convert protobuf transactions to internal format
	for _, protoTx := range protoBlock.Transactions {
		tx := &Transaction{
			ID:        protoTx.Id,
			ChannelID: protoTx.ChannelId,
			Payload:   protoTx.Payload,
			Timestamp: time.Unix(protoTx.Timestamp, 0),
			Identity:  protoTx.Identity,
			Signature: protoTx.Signature,
		}
		channel.Transactions = append(channel.Transactions, tx)
	}

	return nil
}

// checkChannelNeedsSync checks if a channel needs synchronization
func (bs *BlockSynchronizer) checkChannelNeedsSync(channelID string) (bool, error) {
	// For now, we'll use a simple heuristic
	// In a real implementation, we'd query the orderer for the latest block height

	localHeight := bs.blockStorage.GetChannelHeight(channelID)

	// Check if we have recent blocks (within last 5 minutes)
	if localHeight > 0 {
		lastBlock, err := bs.blockStorage.GetBlock(channelID, localHeight-1)
		if err != nil {
			return true, nil // If we can't read the last block, assume we need sync
		}

		// If last block is older than 5 minutes, we might need sync
		if time.Since(lastBlock.Timestamp) > 5*time.Minute {
			return true, nil
		}
	}

	return false, nil
}

// GetSyncStatus returns the current synchronization status
func (bs *BlockSynchronizer) GetSyncStatus() map[string]interface{} {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()

	status := make(map[string]interface{})
	status["is_running"] = bs.isRunning

	channelStatus := make(map[string]interface{})
	channels := bs.peer.channelManager.GetChannelNames()

	for _, channelID := range channels {
		height := bs.blockStorage.GetChannelHeight(channelID)
		needsSync, _ := bs.checkChannelNeedsSync(channelID)

		channelStatus[channelID] = map[string]interface{}{
			"height":     height,
			"needs_sync": needsSync,
		}
	}

	status["channels"] = channelStatus
	return status
}

// ForceSync forces synchronization for a specific channel
func (bs *BlockSynchronizer) ForceSync(ctx context.Context, channelID string) error {
	logger.Infof("Forcing synchronization for channel %s", channelID)

	config := DefaultSyncConfig()
	return bs.syncChannel(ctx, channelID, config)
}
