package orderer

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/ddr4869/minifab/common/logger"
	pb "github.com/ddr4869/minifab/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type OrdererServer struct {
	pb.UnimplementedOrdererServiceServer
	orderer *Orderer
	server  *grpc.Server
	mutex   sync.RWMutex
}

func NewOrdererServer(orderer *Orderer) *OrdererServer {
	return &OrdererServer{
		orderer: orderer,
	}
}

func (s *OrdererServer) SubmitTransaction(ctx context.Context, req *pb.Transaction) (*pb.TransactionResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 트랜잭션을 블록에 추가
	block, err := s.orderer.CreateBlock(req.Payload)
	if err != nil {
		return &pb.TransactionResponse{
			Status:        pb.StatusCode_INTERNAL_ERROR,
			Message:       fmt.Sprintf("Failed to create block: %v", err),
			TransactionId: req.Id,
		}, nil
	}

	return &pb.TransactionResponse{
		Status:        pb.StatusCode_OK,
		Message:       fmt.Sprintf("Transaction %s added to block %d", req.Id, block.Number),
		TransactionId: req.Id,
	}, nil
}

func (s *OrdererServer) GetBlock(ctx context.Context, req *pb.BlockRequest) (*pb.Block, error) {
	logger.Infof("GetBlock request for block %d on channel %s", req.BlockNumber, req.ChannelId)

	// Get block from orderer
	block, err := s.orderer.GetBlock(req.BlockNumber)
	if err != nil {
		logger.Errorf("Failed to get block %d: %v", req.BlockNumber, err)
		return nil, errors.Errorf("block %d not found", req.BlockNumber)
	}

	// Convert to protobuf format
	pbBlock := &pb.Block{
		Number:       block.Number,
		PreviousHash: block.PreviousHash,
		DataHash:     block.Data, // Changed from Data to DataHash
		Timestamp:    block.Timestamp.Unix(),
		ChannelId:    req.ChannelId,
	}

	logger.Infof("Successfully retrieved block %d", req.BlockNumber)
	return pbBlock, nil
}

func (s *OrdererServer) CreateChannel(ctx context.Context, req *pb.ChannelRequest) (*pb.ChannelResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	logger.Infof("[Orderer] Creating channel: %s", req.ChannelName)

	// 채널이 이미 존재하는지 확인
	if _, exists := s.orderer.channels[req.ChannelName]; exists {
		return &pb.ChannelResponse{
			Status:    pb.StatusCode_ALREADY_EXISTS,
			Message:   fmt.Sprintf("Channel %s already exists", req.ChannelName),
			ChannelId: req.ChannelName,
		}, nil
	}

	// 새 채널 생성
	channel := &Channel{
		Name:   req.ChannelName,
		Blocks: make([]*Block, 0),
		MSP:    s.orderer.msp,
	}

	s.orderer.channels[req.ChannelName] = channel

	return &pb.ChannelResponse{
		Status:    pb.StatusCode_OK,
		Message:   fmt.Sprintf("Channel %s created successfully", req.ChannelName),
		ChannelId: req.ChannelName,
	}, nil
}

func (s *OrdererServer) Start(address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return errors.Wrap(err, "failed to listen")
	}

	logger.Infof("Orderer server listening on %s", address)

	s.server = grpc.NewServer()
	pb.RegisterOrdererServiceServer(s.server, s)

	logger.Info("Orderer server started successfully")
	return s.server.Serve(lis)
}

// StartWithContext starts the server with context support for graceful shutdown
func (s *OrdererServer) StartWithContext(ctx context.Context, address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return errors.Wrap(err, "failed to listen")
	}

	logger.Infof("Orderer server listening on %s", address)

	s.server = grpc.NewServer()
	pb.RegisterOrdererServiceServer(s.server, s)

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
