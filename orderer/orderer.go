package orderer

import (
	"sync"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
)

type Orderer struct {
	Mutex sync.RWMutex
	Msp   msp.MSP
	MspID string
}

// NewOrdererWithMSPFiles fabric-ca로 생성된 MSP 파일들을 사용하여 Orderer 생성
func NewOrderer(mspID string, mspPath string) (*Orderer, error) {
	// MSP 파일들로부터 MSP, Identity, PrivateKey 로드
	fabricMSP, err := msp.LoadMSPFromFiles(mspID, mspPath)
	if err != nil {
		logger.Errorf("Failed to create MSP from files: %v", err)
		return nil, err
	}

	logger.Infof("✅ Successfully loaded Orderer MSP from %s", mspPath)
	logger.Info("📋 Orderer Identity Details:")
	logger.Infof("   - ID: %s", fabricMSP.GetSigningIdentity().GetIdentifier().Id)
	logger.Infof("   - MSP ID: %s", fabricMSP.GetSigningIdentity().GetIdentifier().Mspid)

	return &Orderer{
		Msp:   fabricMSP,
		MspID: mspID,
	}, nil
}

func (o *Orderer) GetMSP() msp.MSP {
	o.Mutex.RLock()
	defer o.Mutex.RUnlock()
	return o.Msp
}

// GetMSPID MSP ID 반환
func (o *Orderer) GetMSPID() string {
	o.Mutex.RLock()
	defer o.Mutex.RUnlock()
	return o.MspID
}
