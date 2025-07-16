package peer

import (
	"fmt"
	"sync"
	"time"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
	"github.com/pkg/errors"
)

type Block struct {
	Number       uint64
	PreviousHash []byte
	Data         []byte
	Timestamp    time.Time
}

type Transaction struct {
	ID        string
	ChannelID string
	Payload   []byte
	Timestamp time.Time
	Identity  []byte
	Signature []byte
}

type Channel struct {
	Name         string               `json:"name"`
	Config       *ChannelConfig       `json:"config"`
	GenesisBlock *ChannelGenesisBlock `json:"genesis_block"`
	Transactions []*Transaction       `json:"transactions"`
	State        map[string][]byte    `json:"state"`
	MSP          msp.MSP              `json:"-"`          // JSON 직렬화에서 제외
	MSPConfig    *msp.MSPConfig       `json:"msp_config"` // MSP 설정 정보 저장
}

type Peer struct {
	ID             string
	channelManager *ChannelManager
	transactions   []*Transaction
	mutex          sync.RWMutex
	chaincodePath  string
	msp            msp.MSP
	mspID          string
}

func NewPeer(id string, chaincodePath string, mspID string) *Peer {
	// MSP 인스턴스 생성
	fabricMSP := msp.NewFabricMSP()

	// 기본 MSP 설정
	config := &msp.MSPConfig{
		Name: mspID,
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

	fabricMSP.Setup(config)

	return &Peer{
		ID:             id,
		channelManager: NewChannelManager(),
		transactions:   make([]*Transaction, 0),
		chaincodePath:  chaincodePath,
		msp:            fabricMSP,
		mspID:          mspID,
	}
}

// NewPeerWithMSPFiles fabric-ca로 생성된 MSP 파일들을 사용하여 Peer 생성
func NewPeerWithMSPFiles(id string, chaincodePath string, mspID string, mspPath string) *Peer {
	// MSP 파일들로부터 MSP, Identity, PrivateKey 로드
	fabricMSP, identity, privateKey, err := msp.CreateMSPFromFiles(mspPath, mspID)
	if err != nil {
		logger.Errorf("Failed to create MSP from files: %v", err)
		// 실패 시 기본 MSP 사용
		return NewPeer(id, chaincodePath, mspID)
	}

	logger.Infof("✅ Successfully loaded MSP from %s", mspPath)
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

	return &Peer{
		ID:             id,
		channelManager: NewChannelManager(),
		transactions:   make([]*Transaction, 0),
		chaincodePath:  chaincodePath,
		msp:            fabricMSP,
		mspID:          mspID,
	}
}

func (p *Peer) JoinChannel(channelName string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	channel, err := p.channelManager.GetChannel(channelName)
	if err != nil {
		return errors.Wrap(err, "failed to get channel")
	}

	// 채널용 MSP 생성
	channelMSP := msp.NewFabricMSP()
	config := &msp.MSPConfig{
		Name: fmt.Sprintf("%s.%s", p.mspID, channelName),
		CryptoConfig: &msp.FabricCryptoConfig{
			SignatureHashFamily:            "SHA2",
			IdentityIdentifierHashFunction: "SHA256",
		},
	}
	channelMSP.Setup(config)
	channel.MSP = channelMSP

	return nil
}

func (p *Peer) SubmitTransaction(channelID string, payload []byte) (*Transaction, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	channel, err := p.channelManager.GetChannel(channelID)
	if err != nil {
		return nil, errors.Errorf("channel %s not found", channelID)
	}

	// 임시 Identity와 서명 생성 (실제로는 인증서와 개인키 사용)
	identity := []byte(fmt.Sprintf("peer:%s:%s", p.mspID, p.ID))
	signature := []byte("temp_signature")

	tx := &Transaction{
		ID:        generateTransactionID(),
		ChannelID: channelID,
		Payload:   payload,
		Timestamp: time.Now(),
		Identity:  identity,
		Signature: signature,
	}

	p.transactions = append(p.transactions, tx)
	channel.Transactions = append(channel.Transactions, tx)

	return tx, nil
}

// ValidateTransaction 트랜잭션 검증 (MSP 사용)
func (p *Peer) ValidateTransaction(tx *Transaction) error {
	channel, err := p.channelManager.GetChannel(tx.ChannelID)
	if err != nil {
		return errors.Errorf("channel %s not found", tx.ChannelID)
	}

	fmt.Println("channel", channel)
	// Identity 역직렬화
	identity, err := channel.MSP.DeserializeIdentity(tx.Identity)
	fmt.Println("identity", identity)
	if err != nil {
		return errors.Errorf("failed to deserialize identity: %v", err)
	}

	// Identity 검증
	if err := channel.MSP.ValidateIdentity(identity); err != nil {
		return errors.Errorf("invalid identity: %v", err)
	}

	// 서명 검증
	if err := identity.Verify(tx.Payload, tx.Signature); err != nil {
		return errors.Errorf("signature verification failed: %v", err)
	}

	return nil
}

// GetMSP MSP 인스턴스 반환
func (p *Peer) GetMSP() msp.MSP {
	return p.msp
}

// GetMSPID MSP ID 반환
func (p *Peer) GetMSPID() string {
	return p.mspID
}

// GetChannelManager 채널 관리자 반환
func (p *Peer) GetChannelManager() *ChannelManager {
	return p.channelManager
}

func generateTransactionID() string {
	// 실제 구현에서는 고유한 트랜잭션 ID 생성 로직 구현
	return fmt.Sprintf("tx_%d", time.Now().UnixNano())
}
