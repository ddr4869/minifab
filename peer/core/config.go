package core

import "github.com/ddr4869/minifab/common/msp"

type PeerConfig struct {
	PeerID string
	Msp    msp.MSP
}
