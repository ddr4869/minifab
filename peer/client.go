package peer

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/proto"
)

type OrdererClient struct {
	conn   *grpc.ClientConn
	client proto.OrdererServiceClient
}

func NewOrdererClient(address string) (*OrdererClient, error) {
	logger.Infof("Attempting to connect to orderer at: %s", address)

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to orderer")
	}

	// 연결 상태 확인
	state := conn.GetState()
	if state == connectivity.Ready {
		logger.Infof("Successfully connected to orderer at: %s", address)
	} else {
		logger.Infof("Connection to orderer at %s is %s", address, state.String())
	}

	client := proto.NewOrdererServiceClient(conn)

	return &OrdererClient{
		conn:   conn,
		client: client,
	}, nil
}

func (oc *OrdererClient) SubmitTransaction(tx *Transaction) error {
	if oc.client == nil {
		return errors.New("client not connected")
	}

	// Convert Transaction to proto
	protoTx := &proto.Transaction{
		Id:        tx.ID,
		ChannelId: tx.ChannelID,
		Payload:   tx.Payload,
		Timestamp: tx.Timestamp.Unix(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := oc.client.SubmitTransaction(ctx, protoTx)
	if err != nil {
		return errors.Wrap(err, "failed to submit transaction")
	}

	logger.Infof("Transaction submitted successfully: %s", resp.Message)
	return nil
}

func (oc *OrdererClient) CreateChannel(channelName string) error {
	if oc.client == nil {
		return errors.New("client not connected")
	}

	// gRPC 요청 생성
	req := &proto.ChannelRequest{
		ChannelName: channelName,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := oc.client.CreateChannel(ctx, req)
	if err != nil {
		return errors.Wrap(err, "failed to create channel")
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
