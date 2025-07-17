package client

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
	"github.com/ddr4869/minifab/common/types"
	"github.com/ddr4869/minifab/proto"
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

func (oc *OrdererClient) SubmitTransaction(tx *types.Transaction) error {
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
	return oc.CreateChannelWithProfile(channelName, "testchannel0", "config/configtx.yaml")
}

// CreateChannelWithProfile creates a channel with specified profile from configtx.yaml
func (oc *OrdererClient) CreateChannelWithProfile(channelName, profileName, configTxPath string) error {
	if oc.client == nil {
		return errors.New("client not connected")
	}

	logger.Infof("Creating channel %s with profile %s", channelName, profileName)

	// gRPC 요청 생성 (profile 정보 포함)
	req := &proto.ChannelRequest{
		ChannelName:  channelName,
		ProfileName:  profileName,
		ConfigtxPath: configTxPath,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := oc.client.CreateChannel(ctx, req)
	if err != nil {
		return errors.Wrap(err, "failed to create channel")
	}

	if resp.Status != proto.StatusCode_OK {
		// If channel already exists, that's okay for our use case
		if resp.Status == proto.StatusCode_ALREADY_EXISTS {
			logger.Infof("Channel %s already exists: %s", channelName, resp.Message)
			return nil
		}
		return errors.Errorf("orderer returned error: %s", resp.Message)
	}

	logger.Infof("Channel created successfully: %s", resp.Message)
	return nil
}

func (oc *OrdererClient) Close() error {
	return oc.conn.Close()
}

// GetBlock retrieves a specific block from the orderer
func (oc *OrdererClient) GetBlock(channelID string, blockNumber uint64) (*proto.Block, error) {
	if oc.client == nil {
		return nil, errors.New("client not connected")
	}

	req := &proto.BlockRequest{
		BlockNumber:         blockNumber,
		ChannelId:           channelID,
		IncludeTransactions: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	block, err := oc.client.GetBlock(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block from orderer")
	}

	return block, nil
}

// GetBlockRange retrieves a range of blocks from the orderer
func (oc *OrdererClient) GetBlockRange(channelID string, startBlock, endBlock uint64) ([]*proto.Block, error) {
	if oc.client == nil {
		return nil, errors.New("client not connected")
	}

	req := &proto.BlockRangeRequest{
		ChannelId:           channelID,
		StartBlock:          startBlock,
		EndBlock:            endBlock,
		IncludeTransactions: true,
		MaxBlocks:           100, // Limit to prevent memory issues
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := oc.client.GetBlockRange(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block range from orderer")
	}

	if resp.Status != proto.StatusCode_OK {
		return nil, errors.Errorf("orderer returned error: %s", resp.Message)
	}

	return resp.Blocks, nil
}

// StreamBlocks streams blocks from the orderer starting from a specific block
func (oc *OrdererClient) StreamBlocks(channelID string, startBlock uint64) (<-chan *proto.Block, <-chan error) {
	blockChan := make(chan *proto.Block, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(blockChan)
		defer close(errorChan)

		if oc.client == nil {
			errorChan <- errors.New("client not connected")
			return
		}

		req := &proto.BlockStreamRequest{
			ChannelId:           channelID,
			StartBlock:          startBlock,
			EndBlock:            0, // Stream indefinitely
			IncludeTransactions: true,
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		stream, err := oc.client.StreamBlocks(ctx, req)
		if err != nil {
			errorChan <- errors.Wrap(err, "failed to start block stream")
			return
		}

		for {
			block, err := stream.Recv()
			if err != nil {
				if err.Error() != "EOF" {
					errorChan <- errors.Wrap(err, "error receiving block from stream")
				}
				return
			}

			select {
			case blockChan <- block:
			case <-ctx.Done():
				return
			}
		}
	}()

	return blockChan, errorChan
}

// generateRandomID 랜덤 ID 생성
func generateRandomID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}
