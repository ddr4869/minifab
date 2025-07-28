package server

// import (
// 	"context"
// 	"fmt"
// 	"net"
// 	"sync"
// 	"time"

// 	"github.com/ddr4869/minifab/common/logger"
// 	"github.com/ddr4869/minifab/common/types"
// 	"github.com/ddr4869/minifab/peer/core"
// 	"github.com/ddr4869/minifab/peer/storage"
// 	pb "github.com/ddr4869/minifab/common/proto"
// 	"github.com/pkg/errors"
// 	"google.golang.org/grpc"
// )

// type PeerServer struct {
// 	pb.UnimplementedPeerServiceServer
// 	peer   *core.Peer
// 	server *grpc.Server
// 	mutex  sync.RWMutex
// 	// Block storage for persistence
// 	blockStorage *storage.BlockStorage
// }

// func NewPeerServer(peer *core.Peer) *PeerServer {
// 	return &PeerServer{
// 		peer:         peer,
// 		blockStorage: storage.NewBlockStorage(),
// 	}
// }

// // ProcessBlock handles incoming blocks from orderer
// func (s *PeerServer) ProcessBlock(ctx context.Context, block *pb.Block) (*pb.ProcessBlockResponse, error) {
// 	logger.Infof("[Peer] Processing block %d for channel %s", block.Number, block.ChannelId)

// 	if block == nil {
// 		return &pb.ProcessBlockResponse{
// 			Status:  pb.StatusCode_INVALID_ARGUMENT,
// 			Message: "Block cannot be nil",
// 		}, nil
// 	}

// 	if block.ChannelId == "" {
// 		return &pb.ProcessBlockResponse{
// 			Status:  pb.StatusCode_INVALID_ARGUMENT,
// 			Message: "Channel ID cannot be empty",
// 		}, nil
// 	}

// 	// Validate block structure
// 	if err := s.validateBlock(block); err != nil {
// 		logger.Errorf("Block validation failed: %v", err)
// 		return &pb.ProcessBlockResponse{
// 			Status:  pb.StatusCode_BLOCK_VALIDATION_FAILED,
// 			Message: fmt.Sprintf("Block validation failed: %v", err),
// 		}, nil
// 	}

// 	// Get or create channel
// 	channel, err := s.peer.GetChannelManager().GetChannel(block.ChannelId)
// 	if err != nil {
// 		logger.Errorf("Channel %s not found: %v", block.ChannelId, err)
// 		return &pb.ProcessBlockResponse{
// 			Status:  pb.StatusCode_CHANNEL_NOT_FOUND,
// 			Message: fmt.Sprintf("Channel %s not found", block.ChannelId),
// 		}, nil
// 	}

// 	// Validate transactions in block
// 	validTxCount, invalidTxCount := s.validateTransactions(block, channel)

// 	// Convert protobuf block to internal format
// 	internalBlock := s.convertToInternalBlock(block)

// 	// Persist block atomically
// 	if err := s.blockStorage.StoreBlock(block.ChannelId, internalBlock); err != nil {
// 		logger.Errorf("Failed to persist block %d: %v", block.Number, err)
// 		return &pb.ProcessBlockResponse{
// 			Status:  pb.StatusCode_STORAGE_ERROR,
// 			Message: fmt.Sprintf("Failed to persist block: %v", err),
// 		}, nil
// 	}

// 	// Update channel state
// 	s.mutex.Lock()
// 	channel.Transactions = append(channel.Transactions, s.extractTransactions(block)...)
// 	s.mutex.Unlock()

// 	logger.Infof("Successfully processed block %d: %d valid, %d invalid transactions",
// 		block.Number, validTxCount, invalidTxCount)

// 	return &pb.ProcessBlockResponse{
// 		Status:              pb.StatusCode_OK,
// 		Message:             fmt.Sprintf("Block %d processed successfully", block.Number),
// 		BlockNumber:         block.Number,
// 		ValidTransactions:   validTxCount,
// 		InvalidTransactions: invalidTxCount,
// 	}, nil
// }

// // validateBlock performs basic block structure validation
// func (s *PeerServer) validateBlock(block *pb.Block) error {
// 	if len(block.DataHash) == 0 {
// 		return errors.New("block data hash cannot be empty")
// 	}

// 	if block.Timestamp <= 0 {
// 		return errors.New("block timestamp must be positive")
// 	}

// 	// Validate block sequence (should be consecutive)
// 	if block.Number > 0 {
// 		expectedPrevHash := s.blockStorage.GetLastBlockHash(block.ChannelId)
// 		if expectedPrevHash != nil && string(block.PreviousHash) != string(expectedPrevHash) {
// 			return errors.New("block previous hash mismatch")
// 		}
// 	}

// 	return nil
// }

// // validateTransactions validates all transactions in a block
// func (s *PeerServer) validateTransactions(block *pb.Block, channel *types.Channel) (int32, int32) {
// 	var validCount, invalidCount int32

// 	for _, tx := range block.Transactions {
// 		if err := s.validateTransaction(tx, channel); err != nil {
// 			logger.Warnf("Transaction %s validation failed: %v", tx.Id, err)
// 			invalidCount++
// 		} else {
// 			validCount++
// 		}
// 	}

// 	return validCount, invalidCount
// }

// // validateTransaction validates a single transaction
// func (s *PeerServer) validateTransaction(tx *pb.Transaction, channel *types.Channel) error {
// 	if tx == nil {
// 		return errors.New("transaction cannot be nil")
// 	}

// 	if tx.Id == "" {
// 		return errors.New("transaction ID cannot be empty")
// 	}

// 	if len(tx.Payload) == 0 {
// 		return errors.New("transaction payload cannot be empty")
// 	}

// 	if len(tx.Identity) == 0 {
// 		return errors.New("transaction identity cannot be empty")
// 	}

// 	if len(tx.Signature) == 0 {
// 		return errors.New("transaction signature cannot be empty")
// 	}

// 	// Validate transaction signature using MSP
// 	if channel.MSP != nil {
// 		identity, err := channel.MSP.DeserializeIdentity(tx.Identity)
// 		if err != nil {
// 			return errors.Wrap(err, "failed to deserialize identity")
// 		}

// 		if err := channel.MSP.ValidateIdentity(identity); err != nil {
// 			return errors.Wrap(err, "invalid identity")
// 		}

// 		if err := identity.Verify(tx.Payload, tx.Signature); err != nil {
// 			return errors.Wrap(err, "signature verification failed")
// 		}
// 	}

// 	return nil
// }

// // convertToInternalBlock converts protobuf block to internal block format
// func (s *PeerServer) convertToInternalBlock(pbBlock *pb.Block) *types.Block {
// 	return &types.Block{
// 		Number:       pbBlock.Number,
// 		PreviousHash: pbBlock.PreviousHash,
// 		Data:         pbBlock.DataHash,
// 		Timestamp:    time.Unix(pbBlock.Timestamp, 0),
// 	}
// }

// // extractTransactions converts protobuf transactions to internal format
// func (s *PeerServer) extractTransactions(block *pb.Block) []*types.Transaction {
// 	var transactions []*types.Transaction

// 	for _, pbTx := range block.Transactions {
// 		tx := &types.Transaction{
// 			ID:        pbTx.Id,
// 			ChannelID: pbTx.ChannelId,
// 			Payload:   pbTx.Payload,
// 			Timestamp: time.Unix(pbTx.Timestamp, 0),
// 			Identity:  pbTx.Identity,
// 			Signature: pbTx.Signature,
// 		}
// 		transactions = append(transactions, tx)
// 	}

// 	return transactions
// }

// // SyncBlocks handles block synchronization requests from peers
// func (s *PeerServer) SyncBlocks(req *pb.BlockSyncRequest, stream pb.PeerService_SyncBlocksServer) error {
// 	logger.Infof("[Peer] Syncing blocks for channel %s from block %d", req.ChannelId, req.StartBlock)

// 	if req.ChannelId == "" {
// 		return errors.New("channel ID cannot be empty")
// 	}

// 	// Get blocks from storage starting from requested block
// 	blocks, err := s.blockStorage.GetBlockRange(req.ChannelId, req.StartBlock, req.CurrentHeight)
// 	if err != nil {
// 		logger.Errorf("Failed to get block range: %v", err)
// 		return errors.Wrap(err, "failed to get blocks for sync")
// 	}

// 	// Stream blocks to requesting peer
// 	for _, block := range blocks {
// 		pbBlock := s.convertToProtobufBlock(block, req.ChannelId)
// 		if err := stream.Send(pbBlock); err != nil {
// 			logger.Errorf("Failed to send block %d: %v", block.Number, err)
// 			return errors.Wrap(err, "failed to send block")
// 		}
// 		logger.Debugf("Sent block %d to peer %s", block.Number, req.PeerId)
// 	}

// 	logger.Infof("Successfully synced %d blocks to peer %s", len(blocks), req.PeerId)
// 	return nil
// }

// // convertToProtobufBlock converts internal block to protobuf format
// func (s *PeerServer) convertToProtobufBlock(block *types.Block, channelId string) *pb.Block {
// 	return &pb.Block{
// 		Number:       block.Number,
// 		PreviousHash: block.PreviousHash,
// 		DataHash:     block.Data,
// 		Timestamp:    block.Timestamp.Unix(),
// 		ChannelId:    channelId,
// 		Transactions: []*pb.Transaction{}, // Transactions would be populated from storage if needed
// 	}
// }

// // GetChannelHeight returns the current height of a channel
// func (s *PeerServer) GetChannelHeight(ctx context.Context, req *pb.ChannelHeightRequest) (*pb.ChannelHeightResponse, error) {
// 	if req.ChannelId == "" {
// 		return &pb.ChannelHeightResponse{
// 			Status:  pb.StatusCode_INVALID_ARGUMENT,
// 			Message: "Channel ID cannot be empty",
// 		}, nil
// 	}

// 	height := s.blockStorage.GetChannelHeight(req.ChannelId)
// 	lastBlockHash := s.blockStorage.GetLastBlockHash(req.ChannelId)

// 	return &pb.ChannelHeightResponse{
// 		Status:           pb.StatusCode_OK,
// 		Message:          fmt.Sprintf("Channel %s height: %d", req.ChannelId, height),
// 		Height:           height,
// 		CurrentBlockHash: lastBlockHash,
// 	}, nil
// }

// // NotifyBlockCommit handles block commit notifications
// func (s *PeerServer) NotifyBlockCommit(ctx context.Context, req *pb.BlockCommitNotification) (*pb.BlockCommitResponse, error) {
// 	logger.Infof("[Peer] Received block commit notification for block %d on channel %s",
// 		req.BlockNumber, req.ChannelId)

// 	// Update local state based on commit notification
// 	s.mutex.Lock()
// 	defer s.mutex.Unlock()

// 	// Mark block as committed in storage
// 	if err := s.blockStorage.MarkBlockCommitted(req.ChannelId, req.BlockNumber); err != nil {
// 		logger.Errorf("Failed to mark block %d as committed: %v", req.BlockNumber, err)
// 		return &pb.BlockCommitResponse{
// 			Status:       pb.StatusCode_STORAGE_ERROR,
// 			Message:      fmt.Sprintf("Failed to mark block as committed: %v", err),
// 			Acknowledged: false,
// 		}, nil
// 	}

// 	return &pb.BlockCommitResponse{
// 		Status:       pb.StatusCode_OK,
// 		Message:      fmt.Sprintf("Block %d commit acknowledged", req.BlockNumber),
// 		Acknowledged: true,
// 	}, nil
// }

// // Start starts the peer server
// func (s *PeerServer) Start(address string) error {
// 	lis, err := net.Listen("tcp", address)
// 	if err != nil {
// 		return errors.Wrap(err, "failed to listen")
// 	}

// 	logger.Infof("Peer server listening on %s", address)

// 	s.server = grpc.NewServer()
// 	pb.RegisterPeerServiceServer(s.server, s)

// 	logger.Info("Peer server started successfully")
// 	return s.server.Serve(lis)
// }

// // StartWithContext starts the server with context support for graceful shutdown
// func (s *PeerServer) StartWithContext(ctx context.Context, address string) error {
// 	lis, err := net.Listen("tcp", address)
// 	if err != nil {
// 		return errors.Wrap(err, "failed to listen")
// 	}

// 	logger.Infof("Peer server listening on %s", address)

// 	s.server = grpc.NewServer()
// 	pb.RegisterPeerServiceServer(s.server, s)

// 	// Start server in goroutine
// 	go func() {
// 		if err := s.server.Serve(lis); err != nil {
// 			logger.Errorf("Peer server error: %v", err)
// 		}
// 	}()

// 	// Wait for context cancellation
// 	<-ctx.Done()
// 	logger.Info("Shutting down peer server...")
// 	s.server.GracefulStop()
// 	logger.Info("Peer server shut down complete")

// 	return nil
// }
