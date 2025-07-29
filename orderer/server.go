package orderer

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ddr4869/minifab/common/configtx"
	"github.com/ddr4869/minifab/common/logger"
	pb_common "github.com/ddr4869/minifab/proto/common"
	pb_orderer "github.com/ddr4869/minifab/proto/orderer"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
)

type OrdererServer struct {
	pb_orderer.UnimplementedOrdererServiceServer
	orderer *Orderer
	server  *grpc.Server
	mutex   sync.RWMutex
}

func NewOrdererServer(orderer *Orderer) *OrdererServer {
	return &OrdererServer{
		orderer: orderer,
	}
}

// func (s *OrdererServer) SubmitTransaction(ctx context.Context, req *pb_orderer.Transaction) (*pb_orderer.TransactionResponse, error) {
// 	s.mutex.Lock()
// 	defer s.mutex.Unlock()

// 	// 트랜잭션을 블록에 추가
// 	block, err := s.orderer.CreateBlock(req.Payload)
// 	if err != nil {
// 		return &pb_orderer.TransactionResponse{
// 			Status:        pb_orderer.StatusCode_INTERNAL_ERROR,
// 			Message:       fmt.Sprintf("Failed to create block: %v", err),
// 			TransactionId: req.Id,
// 		}, nil
// 	}

// 	return &pb_orderer.TransactionResponse{
// 		Status:        pb_orderer.StatusCode_OK,
// 		Message:       fmt.Sprintf("Transaction %s added to block %d", req.Id, block.Header.Number),
// 		TransactionId: req.Id,
// 	}, nil
// }

// func (s *OrdererServer) GetBlock(ctx context.Context, req *pb_orderer.BlockRequest) (*pb_orderer.Block, error) {
// 	logger.Infof("GetBlock request for block %d on channel %s", req.BlockNumber, req.ChannelId)

// 	// Convert to protobuf format
// 	pb_ordererBlock := &pb_orderer.Block{
// 		Number:       block.Number,
// 		PreviousHash: block.PreviousHash,
// 		DataHash:     block.Data, // Changed from Data to DataHash
// 		Timestamp:    block.Timestamp.Unix(),
// 		ChannelId:    req.ChannelId,
// 	}

// 	logger.Infof("Successfully retrieved block %d", req.BlockNumber)
// 	return pb_ordererBlock, nil
// }

func (s *OrdererServer) CreateChannel(stream pb_orderer.OrdererService_CreateChannelServer) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		logger.Infof("[Orderer] Received message: %s", msg.Payload.Data)

		var configTx configtx.ConfigTx
		if err := yaml.Unmarshal(msg.Payload.Data, &configTx); err != nil {
			return errors.Wrap(err, "failed to parse configtx YAML")
		}

		appConfig, err := configtx.GetAppChannelProfile(profileName)
		if err != nil {
			return errors.Wrap(err, "failed to convert configtx")
		}
		time.Sleep(3 * time.Second)

		stream.Send(&pb_common.Block{
			Header: &pb_common.BlockHeader{
				Number: 1,
			},
		})
	}

}

// configtx.yaml 경로 설정 (기본값 사용)

// configtx.yaml에서 채널 구성 생성
// channelConfig, err := s.createChannelFromProfile(configTxPath, profileName, req.ChannelName)
// if err != nil {
// 	logger.Errorf("Failed to create channel config from profile: %v", err)
// 	return &pb_orderer.ChannelResponse{
// 		Status:  pb_orderer.StatusCode_CONFIGURATION_ERROR,
// 		Message: fmt.Sprintf("Failed to create channel config: %v", err),
// 	}, nil
// }

// // 채널 구성을 JSON 파일로 저장
// if err := saveChannelConfig(req.ChannelName, channelConfig); err != nil {
// 	logger.Errorf("Failed to save channel config: %v", err)
// 	// 채널은 생성되었지만 설정 저장 실패는 경고로 처리
// }

// logger.Infof("Channel %s created successfully from profile %s", req.ChannelName, profileName)
//}

// BroadcastBlock broadcasts a block to all peers in the network
// func (s *OrdererServer) BroadcastBlock(ctx context.Context, req *pb_orderer.BroadcastRequest) (*pb_orderer.BroadcastResponse, error) {
// 	logger.Infof("[Orderer] Broadcasting block %d to channel %s", req.Block.Number, req.ChannelId)

// 	if req.Block == nil {
// 		return &pb_orderer.BroadcastResponse{
// 			Status:  pb_orderer.StatusCode_INVALID_ARGUMENT,
// 			Message: "Block cannot be nil",
// 		}, nil
// 	}

// 	if req.ChannelId == "" {
// 		return &pb_orderer.BroadcastResponse{
// 			Status:  pb_orderer.StatusCode_INVALID_ARGUMENT,
// 			Message: "Channel ID cannot be empty",
// 		}, nil
// 	}

// 	// Verify channel exists
// 	s.mutex.RLock()
// 	_, exists := s.orderer.channels[req.ChannelId]
// 	s.mutex.RUnlock()

// 	if !exists {
// 		return &pb_orderer.BroadcastResponse{
// 			Status:  pb_orderer.StatusCode_CHANNEL_NOT_FOUND,
// 			Message: fmt.Sprintf("Channel %s not found", req.ChannelId),
// 		}, nil
// 	}

// 	// Broadcast to all specified peer endpoints
// 	var successCount int32
// 	var failedPeers []string

// 	for _, peerEndpoint := range req.PeerEndpoints {
// 		if err := s.broadcastToPeer(ctx, peerEndpoint, req.Block); err != nil {
// 			logger.Errorf("Failed to broadcast to peer %s: %v", peerEndpoint, err)
// 			failedPeers = append(failedPeers, peerEndpoint)
// 		} else {
// 			successCount++
// 			logger.Infof("Successfully broadcasted block %d to peer %s", req.Block.Number, peerEndpoint)
// 		}
// 	}

// 	status := pb_orderer.StatusCode_OK
// 	message := fmt.Sprintf("Block %d broadcasted to %d peers", req.Block.Number, successCount)

// 	if len(failedPeers) > 0 {
// 		status = pb_orderer.StatusCode_INTERNAL_ERROR
// 		message = fmt.Sprintf("Block %d partially broadcasted: %d successful, %d failed",
// 			req.Block.Number, successCount, len(failedPeers))
// 	}

// 	return &pb_orderer.BroadcastResponse{
// 		Status:        status,
// 		Message:       message,
// 		PeersNotified: successCount,
// 		FailedPeers:   failedPeers,
// 	}, nil
// }

// BroadcastToChannel broadcasts a block to all peers in a specific channel
// func (s *OrdererServer) BroadcastToChannel(ctx context.Context, req *pb_orderer.ChannelBroadcastRequest) (*pb_orderer.ChannelBroadcastResponse, error) {
// 	logger.Infof("[Orderer] Broadcasting block %d to channel %s", req.Block.Number, req.ChannelId)

// 	if req.Block == nil {
// 		return &pb_orderer.ChannelBroadcastResponse{
// 			Status:  pb_orderer.StatusCode_INVALID_ARGUMENT,
// 			Message: "Block cannot be nil",
// 		}, nil
// 	}

// 	if req.ChannelId == "" {
// 		return &pb_orderer.ChannelBroadcastResponse{
// 			Status:  pb_orderer.StatusCode_INVALID_ARGUMENT,
// 			Message: "Channel ID cannot be empty",
// 		}, nil
// 	}

// 	// Get channel configuration to find peer endpoints
// 	s.mutex.RLock()
// 	channel, exists := s.orderer.channels[req.ChannelId]
// 	s.mutex.RUnlock()

// 	if !exists {
// 		return &pb_orderer.ChannelBroadcastResponse{
// 			Status:  pb_orderer.StatusCode_CHANNEL_NOT_FOUND,
// 			Message: fmt.Sprintf("Channel %s not found", req.ChannelId),
// 		}, nil
// 	}

// 	// Get peer endpoints from options or use default discovery
// 	var peerEndpoints []string
// 	if req.Options != nil && len(req.Options.TargetPeers) > 0 {
// 		peerEndpoints = req.Options.TargetPeers
// 	} else {
// 		// Use default peer discovery (for now, use hardcoded endpoints)
// 		peerEndpoints = []string{"localhost:7051"} // Default peer endpoint
// 	}

// 	// Broadcast to all peers
// 	var results []*pb_orderer.BroadcastResult
// 	var successCount, failedCount int32

// 	for _, peerEndpoint := range peerEndpoints {
// 		startTime := time.Now()
// 		err := s.broadcastToPeer(ctx, peerEndpoint, req.Block)
// 		responseTime := time.Since(startTime).Milliseconds()

// 		result := &pb_orderer.BroadcastResult{
// 			PeerEndpoint:   peerEndpoint,
// 			ResponseTimeMs: responseTime,
// 		}

// 		if err != nil {
// 			result.Status = pb_orderer.StatusCode_NETWORK_ERROR
// 			result.ErrorMessage = err.Error()
// 			failedCount++
// 			logger.Errorf("Failed to broadcast to peer %s: %v", peerEndpoint, err)
// 		} else {
// 			result.Status = pb_orderer.StatusCode_OK
// 			successCount++
// 			logger.Infof("Successfully broadcasted block %d to peer %s in %dms",
// 				req.Block.Number, peerEndpoint, responseTime)
// 		}

// 		results = append(results, result)
// 	}

// 	// Store block in channel
// 	s.mutex.Lock()
// 	internalBlock := &Block{
// 		Number:       req.Block.Number,
// 		PreviousHash: req.Block.PreviousHash,
// 		Data:         req.Block.DataHash,
// 		Timestamp:    time.Unix(req.Block.Timestamp, 0),
// 	}
// 	channel.Blocks = append(channel.Blocks, internalBlock)
// 	s.mutex.Unlock()

// 	status := pb_orderer.StatusCode_OK
// 	message := fmt.Sprintf("Block %d broadcasted successfully", req.Block.Number)

// 	if failedCount > 0 {
// 		if successCount == 0 {
// 			status = pb_orderer.StatusCode_NETWORK_ERROR
// 			message = fmt.Sprintf("Failed to broadcast block %d to all peers", req.Block.Number)
// 		} else {
// 			status = pb_orderer.StatusCode_INTERNAL_ERROR
// 			message = fmt.Sprintf("Block %d partially broadcasted: %d successful, %d failed",
// 				req.Block.Number, successCount, failedCount)
// 		}
// 	}

// 	return &pb_orderer.ChannelBroadcastResponse{
// 		Status:               status,
// 		Message:              message,
// 		SuccessfulBroadcasts: successCount,
// 		FailedBroadcasts:     failedCount,
// 		Results:              results,
// 	}, nil
// }

// broadcastToPeer sends a block to a specific peer endpoint
// func (s *OrdererServer) broadcastToPeer(ctx context.Context, peerEndpoint string, block *pb_orderer.Block) error {
// 	// Create gRPC connection to peer
// 	conn, err := grpc.NewClient(peerEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
// 	if err != nil {
// 		return errors.Wrap(err, "failed to connect to peer")
// 	}
// 	defer conn.Close()

// 	// Create peer service client
// 	client := pb_orderer.NewPeerServiceClient(conn)

// 	// Set timeout for the broadcast
// 	broadcastCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
// 	defer cancel()

// 	// Send block to peer
// 	resp, err := client.ProcessBlock(broadcastCtx, block)
// 	if err != nil {
// 		return errors.Wrap(err, "failed to send block to peer")
// 	}

// 	if resp.Status != pb_orderer.StatusCode_OK {
// 		return errors.Errorf("peer rejected block: %s", resp.Message)
// 	}

// 	return nil
// }

// func (s *OrdererServer) Start(address string) error {
// 	lis, err := net.Listen("tcp", address)
// 	if err != nil {
// 		return errors.Wrap(err, "failed to listen")
// 	}

// 	logger.Infof("Orderer server listening on %s", address)

// 	s.server = grpc.NewServer()
// 	pb_orderer.RegisterOrdererServiceServer(s.server, s)

// 	logger.Info("Orderer server started successfully")
// 	return s.server.Serve(lis)
// }

// // GetChannelInfo returns information about a channel
// func (s *OrdererServer) GetChannelInfo(ctx context.Context, req *pb_orderer.ChannelInfoRequest) (*pb_orderer.ChannelInfoResponse, error) {
// 	if req.ChannelId == "" {
// 		return &pb_orderer.ChannelInfoResponse{
// 			Status:  pb_orderer.StatusCode_INVALID_ARGUMENT,
// 			Message: "Channel ID cannot be empty",
// 		}, nil
// 	}

// 	s.mutex.RLock()
// 	channel, exists := s.orderer.channels[req.ChannelId]
// 	s.mutex.RUnlock()

// 	if !exists {
// 		return &pb_orderer.ChannelInfoResponse{
// 			Status:  pb_orderer.StatusCode_CHANNEL_NOT_FOUND,
// 			Message: fmt.Sprintf("Channel %s not found", req.ChannelId),
// 		}, nil
// 	}

// 	// Calculate current block hash
// 	var currentBlockHash []byte
// 	if len(channel.Blocks) > 0 {
// 		lastBlock := channel.Blocks[len(channel.Blocks)-1]
// 		currentBlockHash = s.orderer.calculateBlockHash(lastBlock)
// 	}

// 	// Get previous block hash
// 	var previousBlockHash []byte
// 	if len(channel.Blocks) > 1 {
// 		prevBlock := channel.Blocks[len(channel.Blocks)-2]
// 		previousBlockHash = s.orderer.calculateBlockHash(prevBlock)
// 	}

// 	info := &pb_orderer.ChannelInfo{
// 		ChannelId:         req.ChannelId,
// 		Height:            uint64(len(channel.Blocks)),
// 		CurrentBlockHash:  currentBlockHash,
// 		PreviousBlockHash: previousBlockHash,
// 		PeerEndpoints:     []string{"localhost:7051"}, // Default peer endpoints
// 	}

// 	return &pb_orderer.ChannelInfoResponse{
// 		Status:  pb_orderer.StatusCode_OK,
// 		Message: fmt.Sprintf("Channel %s info retrieved", req.ChannelId),
// 		Info:    info,
// 	}, nil
// }

// // StreamBlocks streams blocks to clients
// func (s *OrdererServer) StreamBlocks(req *pb_orderer.BlockStreamRequest, stream pb_orderer.OrdererService_StreamBlocksServer) error {
// 	logger.Infof("[Orderer] Starting block stream for channel %s from block %d", req.ChannelId, req.StartBlock)

// 	if req.ChannelId == "" {
// 		return errors.New("channel ID cannot be empty")
// 	}

// 	s.mutex.RLock()
// 	channel, exists := s.orderer.channels[req.ChannelId]
// 	s.mutex.RUnlock()

// 	if !exists {
// 		return errors.Errorf("channel %s not found", req.ChannelId)
// 	}

// 	// Send existing blocks starting from requested block
// 	s.mutex.RLock()
// 	blocks := channel.Blocks
// 	s.mutex.RUnlock()

// 	for i, block := range blocks {
// 		if uint64(i) >= req.StartBlock {
// 			pb_ordererBlock := &pb_orderer.Block{
// 				Number:       block.Number,
// 				PreviousHash: block.PreviousHash,
// 				DataHash:     block.Data,
// 				Timestamp:    block.Timestamp.Unix(),
// 				ChannelId:    req.ChannelId,
// 				Transactions: []*pb_orderer.Transaction{}, // Would be populated from actual transactions
// 			}

// 			if err := stream.Send(pb_ordererBlock); err != nil {
// 				logger.Errorf("Failed to send block %d: %v", block.Number, err)
// 				return err
// 			}
// 		}
// 	}

// 	// For now, we'll end the stream here
// 	// In a real implementation, we'd keep the stream open for new blocks
// 	logger.Infof("Completed streaming %d blocks for channel %s", len(blocks), req.ChannelId)
// 	return nil
// }

// // GetBlockRange returns a range of blocks
// func (s *OrdererServer) GetBlockRange(ctx context.Context, req *pb_orderer.BlockRangeRequest) (*pb_orderer.BlockRangeResponse, error) {
// 	if req.ChannelId == "" {
// 		return &pb_orderer.BlockRangeResponse{
// 			Status:  pb_orderer.StatusCode_INVALID_ARGUMENT,
// 			Message: "Channel ID cannot be empty",
// 		}, nil
// 	}

// 	s.mutex.RLock()
// 	channel, exists := s.orderer.channels[req.ChannelId]
// 	s.mutex.RUnlock()

// 	if !exists {
// 		return &pb_orderer.BlockRangeResponse{
// 			Status:  pb_orderer.StatusCode_CHANNEL_NOT_FOUND,
// 			Message: fmt.Sprintf("Channel %s not found", req.ChannelId),
// 		}, nil
// 	}

// 	var blocks []*pb_orderer.Block
// 	s.mutex.RLock()
// 	channelBlocks := channel.Blocks
// 	s.mutex.RUnlock()

// 	// Determine the actual range
// 	maxBlocks := req.MaxBlocks
// 	if maxBlocks == 0 {
// 		maxBlocks = 100 // Default limit
// 	}

// 	endBlock := req.EndBlock
// 	if endBlock == 0 || endBlock > uint64(len(channelBlocks)) {
// 		endBlock = uint64(len(channelBlocks))
// 	}

// 	count := int32(0)
// 	for i := req.StartBlock; i < endBlock && count < maxBlocks; i++ {
// 		if int(i) < len(channelBlocks) {
// 			block := channelBlocks[i]
// 			pb_ordererBlock := &pb_orderer.Block{
// 				Number:       block.Number,
// 				PreviousHash: block.PreviousHash,
// 				DataHash:     block.Data,
// 				Timestamp:    block.Timestamp.Unix(),
// 				ChannelId:    req.ChannelId,
// 				Transactions: []*pb_orderer.Transaction{}, // Would be populated from actual transactions
// 			}
// 			blocks = append(blocks, pb_ordererBlock)
// 			count++
// 		}
// 	}

// 	hasMore := endBlock < uint64(len(channelBlocks))
// 	nextBlock := endBlock

// 	return &pb_orderer.BlockRangeResponse{
// 		Status:    pb_orderer.StatusCode_OK,
// 		Message:   fmt.Sprintf("Retrieved %d blocks", len(blocks)),
// 		Blocks:    blocks,
// 		HasMore:   hasMore,
// 		NextBlock: nextBlock,
// 	}, nil
// }

// GetOrdererStatus returns orderer status information
// func (s *OrdererServer) GetOrdererStatus(ctx context.Context, req *pb_orderer.OrdererStatusRequest) (*pb_orderer.OrdererStatusResponse, error) {
// 	s.mutex.RLock()
// 	channelCount := len(s.orderer.channels)
// 	blockCount := s.orderer.GetBlockCount()
// 	s.mutex.RUnlock()

// 	var channels []string
// 	if req.IncludeChannels {
// 		channels = s.orderer.GetChannels()
// 	}

// 	status := &pb_orderer.OrdererStatus{
// 		OrdererId: s.orderer.GetMSPID(),
// 		Endpoint:  "localhost:7050", // Would be configurable
// 		IsLeader:  true,             // Single orderer setup
// 		Channels:  channels,
// 		Metrics: &pb_orderer.OrdererMetrics{
// 			TotalBlocks:    blockCount,
// 			ActiveChannels: int32(channelCount),
// 			ConnectedPeers: 1, // Placeholder
// 		},
// 		UptimeSeconds: 3600, // Placeholder
// 		Version:       "1.0.0",
// 	}

// 	return &pb_orderer.OrdererStatusResponse{
// 		Status:        pb_orderer.StatusCode_OK,
// 		Message:       "Orderer status retrieved successfully",
// 		OrdererStatus: status,
// 	}, nil
// }

// // UpdateChannelConfig updates channel configuration
// func (s *OrdererServer) UpdateChannelConfig(ctx context.Context, req *pb_orderer.ChannelConfigUpdateRequest) (*pb_orderer.ChannelConfigUpdateResponse, error) {
// 	// For now, return not implemented
// 	return &pb_orderer.ChannelConfigUpdateResponse{
// 		Status:  pb_orderer.StatusCode_SERVICE_UNAVAILABLE,
// 		Message: "Channel config update not yet implemented",
// 	}, nil
// }

// // StartWithContext starts the server with context support for graceful shutdown
func (s *OrdererServer) StartWithContext(ctx context.Context, address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return errors.Wrap(err, "failed to listen")
	}

	logger.Infof("Orderer server listening on %s", address)

	s.server = grpc.NewServer()
	pb_orderer.RegisterOrdererServiceServer(s.server, s)

	// Start server in goroutine
	go func() {
		if err := s.server.Serve(lis); err != nil {
			logger.Errorf("Server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	logger.Info("Shutting down orderer server...")
	s.server.GracefulStop()
	logger.Info("Orderer server shut down complete")

	return nil
}

// createChannelFromProfile creates channel configuration from configtx.yaml profile
// func (s *OrdererServer) createChannelFromProfile(configTxPath, profileName, channelName string) (map[string]interface{}, error) {
// 	// configtx.yaml에서 GenesisConfig 생성
// 	genesisConfig, err := CreateGenesisConfigFromConfigTx(configTxPath)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "failed to load configtx.yaml")
// 	}

// 	// 채널 구성 생성
// 	channelConfig := map[string]interface{}{
// 		"channel_id":        channelName,
// 		"profile_name":      profileName,
// 		"consortium":        genesisConfig.ConsortiumName,
// 		"organizations":     make([]map[string]interface{}, 0),
// 		"orderer_endpoints": []string{"localhost:7050"},
// 		"peer_endpoints":    []string{"localhost:7051"},
// 		"policies": map[string]interface{}{
// 			"Readers": "ANY Readers",
// 			"Writers": "ANY Writers",
// 			"Admins":  "ANY Admins",
// 		},
// 		"created_at": time.Now().Format(time.RFC3339),
// 	}

// 	// Peer 조직 정보 추가
// 	for _, org := range genesisConfig.PeerOrgs {
// 		orgInfo := map[string]interface{}{
// 			"name":    org.Name,
// 			"msp_id":  org.ID,
// 			"msp_dir": org.MSPDir,
// 		}
// 		channelConfig["organizations"] = append(channelConfig["organizations"].([]map[string]interface{}), orgInfo)
// 	}

// 	logger.Infof("Created channel config for %s using profile %s", channelName, profileName)
// 	return channelConfig, nil
// }

// saveChannelConfig saves channel configuration to JSON file
func saveChannelConfig(channelName string, config map[string]interface{}) error {
	// channels 디렉토리 생성
	channelsDir := "channels"
	if err := os.MkdirAll(channelsDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create channels directory")
	}

	// JSON 파일로 저장
	fileName := fmt.Sprintf("%s.json", channelName)
	filePath := filepath.Join(channelsDir, fileName)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal channel config")
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return errors.Wrap(err, "failed to write channel config file")
	}

	logger.Infof("Channel config saved to %s", filePath)
	return nil
}
