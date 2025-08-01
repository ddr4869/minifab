package server

import (
	"context"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
	"github.com/ddr4869/minifab/config"
	"github.com/ddr4869/minifab/orderer/channel"
	pb_orderer "github.com/ddr4869/minifab/proto/orderer"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type Orderer struct {
	Mutex         sync.RWMutex
	OrdererConfig *config.OrdererCfg
	ChainSupport  *channel.ChainSupport
	pb_orderer.UnimplementedOrdererServiceServer
	Server *grpc.Server
}

// NewOrdererWithMSPFiles fabric-ca로 생성된 MSP 파일들을 사용하여 Orderer 생성
func NewOrderer(ordererId, mspID, mspPath, ordererAddress, genesisPath string) (*Orderer, error) {
	ordererConfig, err := config.LoadOrdererConfig(ordererId)
	if err != nil {
		logger.Errorf("Failed to load orderer config: %v", err)
		return nil, err
	}

	ordererConfig.Address = ordererAddress
	ordererConfig.MSPID = mspID
	ordererConfig.MSPPath = mspPath
	ordererConfig.GenesisPath = genesisPath

	fabricMSP, err := msp.LoadMSPFromFiles(mspID, mspPath)
	if err != nil {
		logger.Errorf("Failed to create MSP from files: %v", err)
		return nil, err
	}
	ordererConfig.MSP = fabricMSP

	cs := &channel.ChainSupport{
		OrdererConfig: ordererConfig,
	}
	cs.LoadSystemChannelConfig(genesisPath)

	return &Orderer{
		OrdererConfig: ordererConfig,
		ChainSupport:  cs,
		Server:        grpc.NewServer(),
	}, nil
}

func (s *Orderer) Start(address string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal, stopping orderer...")
		cancel()
	}()

	lis, err := net.Listen("tcp", address)
	if err != nil {
		return errors.Wrap(err, "failed to listen")
	}

	logger.Infof("Orderer server listening on %s", address)

	s.Server = grpc.NewServer()
	pb_orderer.RegisterOrdererServiceServer(s.Server, s.ChainSupport)

	go func() {
		if err := s.Server.Serve(lis); err != nil {
			logger.Errorf("Server error: %v", err)
		}
	}()

	<-ctx.Done()
	s.Server.GracefulStop()
	logger.Info("Orderer server stopped gracefully")
	return nil
}
