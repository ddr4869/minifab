package types

import (
	"time"

	"github.com/ddr4869/minifab/common/msp"
)

// Block represents a blockchain block
type Block struct {
	Number       uint64
	PreviousHash []byte
	Data         []byte
	Timestamp    time.Time
}

// Transaction represents a blockchain transaction
type Transaction struct {
	ID        string
	ChannelID string
	Payload   []byte
	Timestamp time.Time
	Identity  []byte
	Signature []byte
}

// Channel represents a blockchain channel
type Channel struct {
	Name         string               `json:"name"`
	Config       *ChannelConfig       `json:"config"`
	GenesisBlock *ChannelGenesisBlock `json:"genesis_block"`
	Transactions []*Transaction       `json:"transactions"`
	State        map[string][]byte    `json:"state"`
	MSP          msp.MSP              `json:"-"`          // JSON 직렬화에서 제외
	MSPConfig    *msp.MSPConfig       `json:"msp_config"` // MSP 설정 정보 저장
}

// ChannelConfig 채널 구성 정보
type ChannelConfig struct {
	ChannelID      string            `json:"channel_id"`
	Consortium     string            `json:"consortium"`
	OrdererAddress string            `json:"orderer_address"`
	Organizations  []string          `json:"organizations"`
	Capabilities   map[string]bool   `json:"capabilities"`
	Policies       map[string]string `json:"policies"`
	Version        uint64            `json:"version"`
}

// ChannelGenesisBlock 채널 생성 블록
type ChannelGenesisBlock struct {
	Number       uint64         `json:"number"`
	PreviousHash []byte         `json:"previous_hash"`
	Data         []byte         `json:"data"`
	Config       *ChannelConfig `json:"config"`
	Timestamp    time.Time      `json:"timestamp"`
}
