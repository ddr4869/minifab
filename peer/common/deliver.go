package common

import (
	"context"
	"fmt"
	"time"

	"github.com/ddr4869/minifab/common/logger"
	pb_common "github.com/ddr4869/minifab/proto/common"
	pb_orderer "github.com/ddr4869/minifab/proto/orderer"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

type OrdererService interface {
	CreateChannel(channelName string) error
	Close() error
}

type OrdererClient struct {
	conn   *grpc.ClientConn
	client pb_orderer.OrdererServiceClient
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

	client := pb_orderer.NewOrdererServiceClient(conn)

	return &OrdererClient{
		conn:   conn,
		client: client,
	}, nil
}

// GetClient returns the internal proto client for direct gRPC calls
func (oc *OrdererClient) GetClient() pb_orderer.OrdererServiceClient {
	return oc.client
}

func (oc *OrdererClient) Send(envelope *pb_common.Envelope) (*pb_orderer.BroadcastResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := oc.client.CreateChannel(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to stream(create channel)")
	}
	err = stream.Send(envelope)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send envelope")
	}

	block, err := stream.Recv()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to receive response")
	}

	// 스트림 종료 의도 명시
	if err := stream.CloseSend(); err != nil {
		logger.Warnf("Failed to close send stream: %v", err)
	}

	if block.Status != pb_common.Status_OK {
		return nil, errors.New(fmt.Sprintf("[%d]failed to create channel", block.Status))
	}
	logger.Infof("✅ FINISH")
	return block, nil
}
