package peer

// Re-export common types for backward compatibility
import "github.com/ddr4869/minifab/common/types"

type (
	Block               = types.Block
	Transaction         = types.Transaction
	Channel             = types.Channel
	ChannelConfig       = types.ChannelConfig
	ChannelGenesisBlock = types.ChannelGenesisBlock
)
