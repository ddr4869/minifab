package msp

import "fmt"

// ChannelMSP 인터페이스
type ChannelMSP interface {
	AddMSP(msp MSP) error
	GetMSP(mspID string) (MSP, error)
	//ValidateIdentity(mspID string, identity Identity) error
	ListMSPs() []string
}

// 기본 구현체
type SimpleChannelMSP struct {
	msps map[string]MSP
}

func NewSimpleChannelMSP() *SimpleChannelMSP {
	return &SimpleChannelMSP{
		msps: make(map[string]MSP),
	}
}

func (c *SimpleChannelMSP) AddMSP(msp MSP) error {
	id := msp.GetSigningIdentity().GetIdentifier().Mspid
	if _, exists := c.msps[id]; exists {
		return fmt.Errorf("MSP %s already exists in channel", id)
	}
	c.msps[id] = msp
	return nil
}

func (c *SimpleChannelMSP) GetMSP(mspID string) (MSP, error) {
	msp, ok := c.msps[mspID]
	if !ok {
		return nil, fmt.Errorf("MSP %s not found in channel", mspID)
	}
	return msp, nil
}

// TODO: 채널 멤버십 검증 로직 추가
// func (c *SimpleChannelMSP) ValidateIdentity(mspID string, identity Identity) error {
// 	msp, err := c.GetMSP(mspID)
// 	if err != nil {
// 		return err
// 	}
// 	return msp.ValidateIdentity(identity)
// }

func (c *SimpleChannelMSP) ListMSPs() []string {
	keys := make([]string, 0, len(c.msps))
	for k := range c.msps {
		keys = append(keys, k)
	}
	return keys
}
