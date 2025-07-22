package peer

import "github.com/ddr4869/minifab/common/types"

// ChannelManager interface to break circular dependency
type ChannelManager interface {
	CreateChannel(channelID string, consortium string, ordererAddress string) error
	GetChannel(channelID string) (*types.Channel, error)
	ListChannels() []string
	GetChannelNames() []string
}
