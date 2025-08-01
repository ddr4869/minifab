package core

import (
	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
	"github.com/ddr4869/minifab/config"
	"github.com/ddr4869/minifab/peer/common"
)

type Peer struct {
	// PeerConfig    *config.Config
	Peer          *config.PeerCfg
	Orderer       *config.OrdererCfg
	Client        *config.ClientCfg
	Channel       *config.ChannelCfg
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
	peerConfig.Client.MSPID = mspId
	peerConfig.Client.MSPPath = mspPath
	peerConfig.Orderer.Address = ordererAddress

	peerMSP, err := msp.LoadMSPFromFiles(peerConfig.Peer.MSPID, peerConfig.Peer.MSPPath)
	if err != nil {
		logger.Errorf("Failed to load MSP from files: %v", err)
		return nil, err
	}
	peerConfig.Peer.MSP = peerMSP

	logger.Infof("✅ Loading client MSP from files: %s", peerConfig.Client.MSPPath)
	clientMSP, err := msp.LoadMSPFromFiles(peerConfig.Client.MSPID, peerConfig.Client.MSPPath)
	if err != nil {
		logger.Errorf("Failed to load MSP from files: %v", err)
		return nil, err
	}
	peerConfig.Client.MSP = clientMSP

	ordererClient, err := common.NewOrdererClient(peerConfig.Orderer.Address)
	if err != nil {
		logger.Errorf("Failed to create orderer client: %v", err)
		return nil, err
	}

	return &Peer{
		Peer:          peerConfig.Peer,
		Orderer:       peerConfig.Orderer,
		Client:        peerConfig.Client,
		Channel:       peerConfig.Channel,
		OrdererClient: ordererClient,
	}, nil
}
