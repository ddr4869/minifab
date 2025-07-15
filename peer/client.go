package peer

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/ddr4869/minifab/common/logger"
	pb "github.com/ddr4869/minifab/common/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

type OrdererClient struct {
	conn   *grpc.ClientConn
	client pb.OrdererServiceClient
}

func NewOrdererClient(address string) (*OrdererClient, error) {
	logger.Infof("Attempting to connect to orderer at: %s", address)

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to orderer: %v", err)
	}

	// 연결 상태 확인
	state := conn.GetState()
	if state == connectivity.Ready {
		logger.Infof("Successfully connected to orderer at: %s", address)
	} else {
		logger.Infof("Connection to orderer at %s is %s", address, state.String())
	}

	client := pb.NewOrdererServiceClient(conn)

	return &OrdererClient{
		conn:   conn,
		client: client,
	}, nil
}

func (oc *OrdererClient) SubmitTransaction(tx *Transaction) error {
	// gRPC 요청 생성
	req := &pb.Transaction{
		Id:        tx.ID,
		ChannelId: tx.ChannelID,
		Payload:   tx.Payload,
		Timestamp: tx.Timestamp.Unix(),
	}

	// gRPC 호출
	resp, err := oc.client.SubmitTransaction(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to submit transaction: %v", err)
	}

	logger.Infof("Transaction submitted successfully: %s", resp.Message)
	return nil
}

func (oc *OrdererClient) CreateChannel(channelName string) error {
	// gRPC 요청 생성
	req := &pb.ChannelRequest{
		ChannelName: channelName,
	}

	// gRPC 호출
	resp, err := oc.client.CreateChannel(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to create channel: %v", err)
	}

	logger.Infof("Channel created successfully: %s", resp.Message)
	return nil
}

func (oc *OrdererClient) Close() error {
	return oc.conn.Close()
}

// generateRandomID 랜덤 ID 생성
func generateRandomID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}
