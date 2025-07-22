package common

import (
	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/types"
	"github.com/ddr4869/minifab/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

type OrdererService interface {
	CreateChannel(channelName string) error
	CreateChannelWithProfile(channelName, profileName, configTxPath string) error
	SubmitTransaction(tx *types.Transaction) error
	Close() error
}

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
