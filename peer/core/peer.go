package core

import (
	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
	"github.com/ddr4869/minifab/config"
	"github.com/ddr4869/minifab/peer/common"
)

type Peer struct {
	PeerConfig    *config.Config
	Msp           msp.MSP
	OrdererClient *common.OrdererClient
}

func NewPeer(peerId, mspId, mspPath, ordererAddress string) (*Peer, error) {
	// MSP 파일들로부터 MSP, Identity, PrivateKey 로드
	logger.Infof("✅ Creating peer with ID: %s, MSP ID: %s, MSP Path: %s, Orderer Address: %s", peerId, mspId, mspPath, ordererAddress)

	peerConfig, err := config.LoadPeerConfig(peerId)
	if err != nil {
		logger.Errorf("Failed to load peer config: %v", err)
		return nil, err
	}
	peerConfig.PrintConfig()
	peerConfig.Peer.MSPID = mspId
	peerConfig.Peer.MSPPath = mspPath
	peerConfig.Peer.Address = ordererAddress

	fabricMSP, err := msp.LoadMSPFromFiles(mspId, mspPath)
	if err != nil {
		logger.Errorf("Failed to create MSP from files: %v", err)
		return nil, err
	}

	logger.Infof("✅ Successfully loaded MSP from %s", mspPath)

	ordererClient, err := common.NewOrdererClient(ordererAddress)
	if err != nil {
		logger.Errorf("Failed to create orderer client: %v", err)
		return nil, err
	}

	return &Peer{
		PeerConfig:    peerConfig,
		Msp:           fabricMSP,
		OrdererClient: ordererClient,
	}, nil
}
