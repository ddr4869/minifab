package configtx

// Consortiums represents consortium configurations
type Consortiums struct {
	SampleConsortium Consortium `yaml:"SampleConsortium"`
}

// Consortium represents a consortium configuration
type Consortium struct {
	Organizations []interface{} `yaml:"Organizations"`
}

// GenesisConfig represents the configuration for generating a genesis block
// This mirrors the structure from orderer/genesis.go but is defined here to avoid circular imports
type GenesisConfig struct {
	NetworkName    string                   `json:"network_name"`
	ConsortiumName string                   `json:"consortium_name"`
	OrdererOrgs    []*OrganizationConfig    `json:"orderer_orgs"`
	PeerOrgs       []*OrganizationConfig    `json:"peer_orgs"`
	SystemChannel  *SystemChannelConfig     `json:"system_channel"`
	Policies       map[string]*PolicyConfig `json:"policies"`
	BatchSize      *BatchSizeConfig         `json:"batch_size"`
	BatchTimeout   string                   `json:"batch_timeout"`
}

// SystemChannelConfig represents system channel configuration
type SystemChannelConfig struct {
	Name       string                   `json:"name"`
	Consortium string                   `json:"consortium"`
	Policies   map[string]*PolicyConfig `json:"policies"`
}

// PolicyConfig represents policy configuration
type PolicyConfig struct {
	Type string `json:"type"`
	Rule string `json:"rule"`
}

// BatchSizeConfig represents batch size configuration
type BatchSizeConfig struct {
	MaxMessageCount   uint32 `json:"max_message_count"`
	AbsoluteMaxBytes  uint32 `json:"absolute_max_bytes"`
	PreferredMaxBytes uint32 `json:"preferred_max_bytes"`
}
