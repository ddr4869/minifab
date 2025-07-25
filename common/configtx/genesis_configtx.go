package configtx

type GenesisConfig struct {
	NetworkName   string                `json:"network_name"`   // Name of the blockchain network
	OrdererOrgs   []*OrganizationConfig `json:"orderer_orgs"`   // Orderer organizations configuration
	SystemChannel *SystemChannelConfig  `json:"system_channel"` // System channel configuration
	BatchSize     *BatchSizeConfig      `json:"batch_size"`     // Transaction batching configuration
	BatchTimeout  string                `json:"batch_timeout"`  // Batch timeout duration
}

// OrganizationConfig defines the configuration for a blockchain organization.
// It contains MSP settings and policies for either orderer or peer organizations.
type OrganizationConfig struct {
	Name    string `json:"name"`     // Human-readable organization name
	ID      string `json:"id"`       // MSP identifier for the organization
	MSPDir  string `json:"msp_dir"`  // Path to MSP certificate directory
	MSPType string `json:"msp_type"` // MSP implementation type (e.g., "bccsp")
}

// SystemChannelConfig defines the configuration for the system channel.
// The system channel is used for network-wide configuration and consortium management.
type SystemChannelConfig struct {
	BatchSize    *BatchSizeConfig `json:"batch_size"`    // Transaction batching configuration
	BatchTimeout string           `json:"batch_timeout"` // Batch timeout duration
}

type BatchSizeConfig struct {
	MaxMessageCount   uint32 `json:"max_message_count"`   // Maximum number of transactions per block
	AbsoluteMaxBytes  uint32 `json:"absolute_max_bytes"`  // Hard limit on block size in bytes
	PreferredMaxBytes uint32 `json:"preferred_max_bytes"` // Preferred block size in bytes
}
