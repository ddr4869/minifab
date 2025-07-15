package orderer

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/ddr4869/minifab/common/logger"
	pb "github.com/ddr4869/minifab/common/proto"
	"google.golang.org/grpc"
)

type OrdererServer struct {
	pb.UnimplementedOrdererServiceServer
	orderer *Orderer
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
			Success: false,
			Message: fmt.Sprintf("Failed to create block: %v", err),
		}, nil
	}

	return &pb.TransactionResponse{
		Success: true,
		Message: fmt.Sprintf("Transaction %s added to block %d", req.Id, block.Number),
	}, nil
}

func (s *OrdererServer) GetBlock(ctx context.Context, req *pb.BlockRequest) (*pb.Block, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// 요청된 블록 번호에 해당하는 블록 반환
	if int(req.BlockNumber) >= len(s.orderer.blocks) {
		return nil, fmt.Errorf("block %d not found", req.BlockNumber)
	}

	block := s.orderer.blocks[req.BlockNumber]
	return &pb.Block{
		Number:       block.Number,
		PreviousHash: block.PreviousHash,
		Data:         block.Data,
		Timestamp:    block.Timestamp.Unix(),
	}, nil
}

func (s *OrdererServer) CreateChannel(ctx context.Context, req *pb.ChannelRequest) (*pb.ChannelResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	logger.Infof("[Orderer] Creating channel: %s", req.ChannelName)

	// 채널이 이미 존재하는지 확인
	if _, exists := s.orderer.channels[req.ChannelName]; exists {
		return &pb.ChannelResponse{
			Success: false,
			Message: fmt.Sprintf("Channel %s already exists", req.ChannelName),
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
		Success: true,
		Message: fmt.Sprintf("Channel %s created successfully", req.ChannelName),
	}, nil
}

func (s *OrdererServer) Start(address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	server := grpc.NewServer()
	pb.RegisterOrdererServiceServer(server, s)

	logger.Infof("Orderer server listening on %s", address)
	return server.Serve(lis)
}

// StartWithContext starts the server with context support for graceful shutdown
func (s *OrdererServer) StartWithContext(ctx context.Context, address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	server := grpc.NewServer()
	pb.RegisterOrdererServiceServer(server, s)

	// Channel to capture server errors
	errChan := make(chan error, 1)

	// Start server in goroutine
	go func() {
		logger.Infof("Orderer server listening on %s", address)
		if err := server.Serve(lis); err != nil {
			errChan <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		logger.Info("Shutting down orderer server...")
		server.GracefulStop()
		return nil
	case err := <-errChan:
		return err
	}
}
