package peer

// ChannelManager interface to break circular dependency
type ChannelManager interface {
	CreateChannel(channelID string, consortium string, ordererAddress string) error
	GetChannel(channelID string) (*Channel, error)
	ListChannels() []string
	GetChannelNames() []string
}
