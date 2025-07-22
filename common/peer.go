package common

import (
	"sync"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
	"github.com/ddr4869/minifab/peer/client"
	"github.com/pkg/errors"
)

// PeerClient는 peer 클라이언트 역할을 하는 구조체입니다
type PeerClient struct {
	ID            string
	mutex         sync.RWMutex
	ChaincodePath string
	MSP           msp.MSP
	MSPID         string
	OrdererClient client.OrdererService
}

// PeerConfig는 PeerClient 생성을 위한 설정 구조체입니다
type PeerConfig struct {
	ID             string
	ChaincodePath  string
	MSPID          string
	MSPPath        string
	OrdererAddress string
}

// NewPeerClient는 새로운 PeerClient 인스턴스를 생성합니다
func NewPeerClient(config *PeerConfig) (*PeerClient, error) {
	// Orderer 클라이언트 생성
	ordererClient, err := client.NewOrdererClient(config.OrdererAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create orderer client")
	}

	// MSP 인스턴스 생성
	fabricMSP := msp.NewFabricMSP()

	// 기본 MSP 설정
	mspConfig := &msp.MSPConfig{
		Name: config.MSPID,
		CryptoConfig: &msp.FabricCryptoConfig{
			SignatureHashFamily:            "SHA2",
			IdentityIdentifierHashFunction: "SHA256",
		},
		NodeOUs: &msp.FabricNodeOUs{
			Enable: true,
			PeerOUIdentifier: &msp.FabricOUIdentifier{
				OrganizationalUnitIdentifier: "peer",
			},
		},
	}

	if err := fabricMSP.Setup(mspConfig); err != nil {
		return nil, errors.Wrap(err, "failed to setup MSP")
	}

	return &PeerClient{
		ID:            config.ID,
		ChaincodePath: config.ChaincodePath,
		MSP:           fabricMSP,
		MSPID:         config.MSPID,
		OrdererClient: ordererClient,
	}, nil
}

// NewPeerClientWithMSPFiles는 MSP 파일들을 사용하여 PeerClient를 생성합니다
func NewPeerClientWithMSPFiles(config *PeerConfig) (*PeerClient, error) {
	// Orderer 클라이언트 생성
	ordererClient, err := client.NewOrdererClient(config.OrdererAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create orderer client")
	}

	// MSP 파일들로부터 MSP, Identity, PrivateKey 로드
	fabricMSP, identity, privateKey, err := msp.CreateMSPFromFiles(config.MSPPath, config.MSPID)
	if err != nil {
		logger.Errorf("Failed to create MSP from files: %v", err)
		// 실패 시 기본 MSP 사용
		return NewPeerClient(config)
	}

	logger.Infof("✅ Successfully loaded MSP from %s", config.MSPPath)
	logger.Info("📋 Identity Details:")
	logger.Infof("   - ID: %s", identity.GetIdentifier().Id)
	logger.Infof("   - MSP ID: %s", identity.GetMSPIdentifier())

	// 조직 단위 정보 출력
	ous := identity.GetOrganizationalUnits()
	if len(ous) > 0 {
		logger.Info("   - Organizational Units:")
		for _, ou := range ous {
			logger.Infof("     * %s", ou.OrganizationalUnitIdentifier)
		}
	}

	// privateKey는 나중에 사용할 수 있도록 저장 (현재는 로그만 출력)
	if privateKey != nil {
		logger.Info("🔑 Private key loaded successfully")
	}

	return &PeerClient{
		ID:            config.ID,
		ChaincodePath: config.ChaincodePath,
		MSP:           fabricMSP,
		MSPID:         config.MSPID,
		OrdererClient: ordererClient,
	}, nil
}

// GetID는 피어 ID를 반환합니다
func (pc *PeerClient) GetID() string {
	return pc.ID
}

// GetMSPID는 MSP ID를 반환합니다
func (pc *PeerClient) GetMSPID() string {
	return pc.MSPID
}

// GetChaincodePath는 체인코드 경로를 반환합니다
func (pc *PeerClient) GetChaincodePath() string {
	return pc.ChaincodePath
}

// GetMSP는 MSP 인스턴스를 반환합니다
func (pc *PeerClient) GetMSP() msp.MSP {
	return pc.MSP
}

// GetOrdererClient는 orderer 클라이언트를 반환합니다
func (pc *PeerClient) GetOrdererClient() client.OrdererService {
	return pc.OrdererClient
}

// Close는 PeerClient의 리소스를 정리합니다
func (pc *PeerClient) Close() error {
	if pc.OrdererClient != nil {
		return pc.OrdererClient.Close()
	}
	return nil
}

// GetInfo는 피어 클라이언트의 기본 정보를 반환합니다
func (pc *PeerClient) GetInfo() map[string]interface{} {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()

	return map[string]interface{}{
		"id":             pc.ID,
		"msp_id":         pc.MSPID,
		"chaincode_path": pc.ChaincodePath,
	}
}

// IsConnected는 orderer 클라이언트 연결 상태를 확인합니다
func (pc *PeerClient) IsConnected() bool {
	return pc.OrdererClient != nil
}
