package configtx

// import "github.com/pkg/errors"

// type GenesisConfig struct {
// 	NetworkName   string                `json:"network_name"`   // Name of the blockchain network (profile name)
// 	OrdererOrg    *OrganizationConfig   `json:"orderer_org"`    // Single orderer organization configuration
// 	Consortiums   []*OrganizationConfig `json:"consortiums"`    // Consortium organizations configuration
// 	SystemChannel *SystemChannelConfig  `json:"system_channel"` // System channel configuration
// }

// // OrganizationConfig defines the configuration for a blockchain organization.
// // It contains MSP settings and policies for either orderer or peer organizations.
// type OrganizationConfig struct {
// 	Name    string `json:"name"`     // Human-readable organization name
// 	ID      string `json:"id"`       // MSP identifier for the organization
// 	MSPDir  string `json:"msp_dir"`  // Path to MSP certificate directory
// 	MSPType string `json:"msp_type"` // MSP implementation type (e.g., "bccsp")
// }

// // SystemChannelConfig defines the configuration for the system channel.
// // The system channel is used for network-wide configuration and consortium management.
// type SystemChannelConfig struct {
// 	Name         string             `json:"name"`          // System channel name
// 	BatchSize    *BatchSizeConfig   `json:"batch_size"`    // Transaction batching configuration
// 	BatchTimeout string             `json:"batch_timeout"` // Batch timeout duration
// 	Policies     map[string]*Policy `json:"policies"`      // Channel policies
// }

// type BatchSizeConfig struct {
// 	MaxMessageCount   uint32 `json:"max_message_count"`   // Maximum number of transactions per block
// 	AbsoluteMaxBytes  uint32 `json:"absolute_max_bytes"`  // Hard limit on block size in bytes
// 	PreferredMaxBytes uint32 `json:"preferred_max_bytes"` // Preferred block size in bytes
// }

// // Policy represents a policy configuration
// type Policy struct {
// 	Type string      `json:"type"` // Policy type (e.g., "ImplicitMeta", "Signature")
// 	Rule interface{} `json:"rule"` // Policy rule
// }

// // ImplicitMetaRule represents an ImplicitMeta policy rule
// type ImplicitMetaRule struct {
// 	Rule      string `json:"rule"`       // Rule type (ANY, MAJORITY, ALL)
// 	SubPolicy string `json:"sub_policy"` // Sub-policy name
// }

// // Constants for policy types and rules
// const (
// 	PolicyTypeImplicitMeta = "ImplicitMeta"
// 	PolicyTypeSignature    = "Signature"
// 	PolicyRuleAny          = "ANY"
// 	PolicyRuleMajority     = "MAJORITY"
// 	PolicyRuleAll          = "ALL"

// 	// MSP types
// 	MSPTypeBCCSP = "bccsp"

// 	// Default values
// 	DefaultBatchTimeout  = "200ms"
// 	DefaultSystemChannel = "system-channel"
// )

// // ValidateGenesisConfig validates the genesis configuration
// func ValidateGenesisConfig(config *GenesisConfig) error {
// 	if config == nil {
// 		return errors.New("genesis config cannot be nil")
// 	}

// 	if config.OrdererOrg == nil {
// 		return errors.New("orderer organization is required")
// 	}

// 	return nil
// }
