package orderer

import (
	"crypto/sha256"
	"encoding/json"
	"time"

	"github.com/ddr4869/minifab/common/msp"
	"github.com/pkg/errors"
)

// Constants for genesis block configuration
const (
	// Default values
	DefaultNetworkName    = "CustomFabricNetwork"
	DefaultConsortiumName = "SampleConsortium"
	DefaultSystemChannel  = "system-channel"
	DefaultBatchTimeout   = "2s"
	DefaultConsensusType  = "solo"

	// Capabilities
	CapabilityV2_0 = "V2_0"

	// Hash algorithms
	HashAlgorithmSHA256 = "SHA256"
	HashFamilySHA2      = "SHA2"

	// Policy types
	PolicyTypeSignature    = "Signature"
	PolicyTypeImplicitMeta = "ImplicitMeta"

	// Roles
	RoleMember = "MEMBER"
	RoleAdmin  = "ADMIN"

	// Policy rules
	PolicyRuleAny      = "ANY"
	PolicyRuleMajority = "MAJORITY"

	// MSP types
	MSPTypeBCCSP = "bccsp"

	// Organizational units
	OUClient  = "client"
	OUPeer    = "peer"
	OUAdmin   = "admin"
	OUOrderer = "orderer"

	// Default batch size limits
	DefaultMaxMessageCount   = 100
	DefaultAbsoluteMaxBytes  = 10 * 1024 * 1024 // 10MB
	DefaultPreferredMaxBytes = 2 * 1024 * 1024  // 2MB
)

// GenesisConfig defines the configuration for generating a genesis block.
// It contains all necessary information to bootstrap a Hyperledger Fabric network
// including organization configurations, policies, and consensus parameters.
type GenesisConfig struct {
	NetworkName    string                `json:"network_name"`    // Name of the blockchain network
	ConsortiumName string                `json:"consortium_name"` // Name of the consortium
	OrdererOrgs    []*OrganizationConfig `json:"orderer_orgs"`    // Orderer organizations configuration
	PeerOrgs       []*OrganizationConfig `json:"peer_orgs"`       // Peer organizations configuration
	SystemChannel  *SystemChannelConfig  `json:"system_channel"`  // System channel configuration
	Capabilities   map[string]bool       `json:"capabilities"`    // Network capabilities
	Policies       map[string]*Policy    `json:"policies"`        // Channel-level policies
	BatchSize      *BatchSizeConfig      `json:"batch_size"`      // Transaction batching configuration
	BatchTimeout   string                `json:"batch_timeout"`   // Batch timeout duration
}

// OrganizationConfig defines the configuration for a blockchain organization.
// It contains MSP settings and policies for either orderer or peer organizations.
type OrganizationConfig struct {
	Name     string             `json:"name"`     // Human-readable organization name
	ID       string             `json:"id"`       // MSP identifier for the organization
	MSPDir   string             `json:"msp_dir"`  // Path to MSP certificate directory
	MSPType  string             `json:"msp_type"` // MSP implementation type (e.g., "bccsp")
	Policies map[string]*Policy `json:"policies"` // Organization-specific policies
}

// SystemChannelConfig defines the configuration for the system channel.
// The system channel is used for network-wide configuration and consortium management.
type SystemChannelConfig struct {
	Name         string             `json:"name"`         // System channel name
	Consortium   string             `json:"consortium"`   // Associated consortium name
	Capabilities map[string]bool    `json:"capabilities"` // Enabled capabilities for system channel
	Policies     map[string]*Policy `json:"policies"`     // System channel policies
}

// Policy defines access control policies for various network operations.
// Policies can be signature-based or implicit meta policies.
type Policy struct {
	Type string `json:"type"` // Policy type: "Signature" or "ImplicitMeta"
	Rule any    `json:"rule"` // Policy rule definition (varies by type)
}

// ImplicitMetaRule defines the structure for implicit meta policy rules
type ImplicitMetaRule struct {
	Rule      string `json:"rule"`       // Policy rule (ANY, MAJORITY, etc.)
	SubPolicy string `json:"sub_policy"` // Sub-policy name
}

// SignatureRule defines the structure for signature policy rules
type SignatureRule struct {
	Identities []PolicyIdentity `json:"identities"`
	Rule       NOutOfRule       `json:"rule"`
}

// PolicyIdentity defines a policy identity
type PolicyIdentity struct {
	Principal               PolicyPrincipal `json:"principal"`
	PrincipalClassification string          `json:"principal_classification"`
}

// PolicyPrincipal defines the principal for an identity
type PolicyPrincipal struct {
	MSPIdentifier string `json:"msp_identifier"`
	Role          string `json:"role"`
}

// NOutOfRule defines n-out-of rule structure
type NOutOfRule struct {
	NOutOf NOutOfSpec `json:"n_out_of"`
}

// NOutOfSpec defines the n-out-of specification
type NOutOfSpec struct {
	N     int              `json:"n"`
	Rules []map[string]any `json:"rules"`
}

// BatchSizeConfig defines transaction batching parameters for block creation.
// These settings control how transactions are grouped into blocks.
type BatchSizeConfig struct {
	MaxMessageCount   uint32 `json:"max_message_count"`   // Maximum number of transactions per block
	AbsoluteMaxBytes  uint32 `json:"absolute_max_bytes"`  // Hard limit on block size in bytes
	PreferredMaxBytes uint32 `json:"preferred_max_bytes"` // Preferred block size in bytes
}

// GenesisBlock represents the first block in a blockchain network
type GenesisBlock struct {
	Header *BlockHeader `json:"header"`
	Data   *BlockData   `json:"data"`
}

// BlockHeader contains metadata for a blockchain block
type BlockHeader struct {
	Number       uint64 `json:"number"`
	PreviousHash []byte `json:"previous_hash"`
	DataHash     []byte `json:"data_hash"`
	Timestamp    int64  `json:"timestamp"`
}

// BlockData contains the actual data payload of a block
type BlockData struct {
	ConfigTx *ConfigTransaction `json:"config_tx"`
}

// ConfigTransaction represents a configuration transaction
type ConfigTransaction struct {
	ChannelID    string             `json:"channel_id"`
	ConfigUpdate *ConfigUpdate      `json:"config_update"`
	Signatures   []*ConfigSignature `json:"signatures"`
}

// ConfigUpdate represents changes to channel configuration
type ConfigUpdate struct {
	ChannelID string                  `json:"channel_id"`
	ReadSet   map[string]*ConfigGroup `json:"read_set"`
	WriteSet  map[string]*ConfigGroup `json:"write_set"`
}

// ConfigGroup represents a hierarchical configuration group
type ConfigGroup struct {
	Version   uint64                  `json:"version"`
	Groups    map[string]*ConfigGroup `json:"groups"`
	Values    map[string]*ConfigValue `json:"values"`
	Policies  map[string]*Policy      `json:"policies"`
	ModPolicy string                  `json:"mod_policy"`
}

// ConfigValue represents a configuration value with versioning
type ConfigValue struct {
	Version   uint64 `json:"version"`
	Value     any    `json:"value"`
	ModPolicy string `json:"mod_policy"`
}

// ConfigSignature represents a cryptographic signature on configuration
type ConfigSignature struct {
	SignatureHeader []byte `json:"signature_header"`
	Signature       []byte `json:"signature"`
}

// GenesisBlockGeneratorInterface defines the interface for genesis block generation
type GenesisBlockGeneratorInterface interface {
	GenerateGenesisBlock() (*GenesisBlock, error)
}

// GenesisBlockGenerator generates genesis blocks for blockchain networks
type GenesisBlockGenerator struct {
	config *GenesisConfig
}

// NewGenesisBlockGenerator creates a new genesis block generator
func NewGenesisBlockGenerator(config *GenesisConfig) (*GenesisBlockGenerator, error) {
	if config == nil {
		return nil, errors.New("genesis config cannot be nil")
	}

	if err := validateGenesisConfig(config); err != nil {
		return nil, errors.Wrap(err, "invalid genesis config")
	}

	return &GenesisBlockGenerator{
		config: config,
	}, nil
}

// validateGenesisConfig validates the genesis configuration
func validateGenesisConfig(config *GenesisConfig) error {
	if config.NetworkName == "" {
		return errors.New("network name cannot be empty")
	}

	if config.ConsortiumName == "" {
		return errors.New("consortium name cannot be empty")
	}

	if len(config.OrdererOrgs) == 0 {
		return errors.New("at least one orderer organization is required")
	}

	if config.SystemChannel == nil {
		return errors.New("system channel configuration is required")
	}

	if config.BatchSize == nil {
		return errors.New("batch size configuration is required")
	}

	// Validate batch size limits
	if config.BatchSize.MaxMessageCount == 0 {
		return errors.New("max message count must be greater than 0")
	}

	if config.BatchSize.AbsoluteMaxBytes == 0 {
		return errors.New("absolute max bytes must be greater than 0")
	}

	if config.BatchSize.PreferredMaxBytes > config.BatchSize.AbsoluteMaxBytes {
		return errors.New("preferred max bytes cannot exceed absolute max bytes")
	}

	// Validate organizations
	for i, org := range config.OrdererOrgs {
		if err := validateOrganizationConfig(org); err != nil {
			return errors.Wrapf(err, "invalid orderer org at index %d", i)
		}
	}

	for i, org := range config.PeerOrgs {
		if err := validateOrganizationConfig(org); err != nil {
			return errors.Wrapf(err, "invalid peer org at index %d", i)
		}
	}

	return nil
}

// validateOrganizationConfig validates organization configuration
func validateOrganizationConfig(org *OrganizationConfig) error {
	if org == nil {
		return errors.New("organization config cannot be nil")
	}

	if org.Name == "" {
		return errors.New("organization name cannot be empty")
	}

	if org.ID == "" {
		return errors.New("organization ID cannot be empty")
	}

	if org.MSPDir == "" {
		return errors.New("MSP directory cannot be empty")
	}

	return nil
}

// GenerateGenesisBlock generates the genesis block for the blockchain network
func (g *GenesisBlockGenerator) GenerateGenesisBlock() (*GenesisBlock, error) {
	configTx, err := g.createSystemChannelConfigTx()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create system channel config tx")
	}

	blockData := &BlockData{ConfigTx: configTx}

	header, err := g.createGenesisBlockHeader(blockData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create genesis block header")
	}

	return &GenesisBlock{
		Header: header,
		Data:   blockData,
	}, nil
}

// createGenesisBlockHeader creates the header for the genesis block
func (g *GenesisBlockGenerator) createGenesisBlockHeader(blockData *BlockData) (*BlockHeader, error) {
	dataBytes, err := json.Marshal(blockData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal block data")
	}

	dataHash := sha256.Sum256(dataBytes)

	return &BlockHeader{
		Number:       0,
		PreviousHash: nil, // Genesis block has no previous hash
		DataHash:     dataHash[:],
		Timestamp:    time.Now().Unix(),
	}, nil
}

// createSystemChannelConfigTx creates the system channel configuration transaction
func (g *GenesisBlockGenerator) createSystemChannelConfigTx() (*ConfigTransaction, error) {
	// Create channel configuration group
	channelGroup, err := g.createChannelConfigGroup()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create channel config group")
	}

	// Create configuration update
	configUpdate := &ConfigUpdate{
		ChannelID: g.config.SystemChannel.Name,
		ReadSet:   make(map[string]*ConfigGroup),
		WriteSet: map[string]*ConfigGroup{
			"Channel": channelGroup,
		},
	}

	// Create configuration transaction
	configTx := &ConfigTransaction{
		ChannelID:    g.config.SystemChannel.Name,
		ConfigUpdate: configUpdate,
		Signatures:   make([]*ConfigSignature, 0), // Genesis block has no signatures
	}

	return configTx, nil
}

// createChannelConfigGroup creates the channel configuration group
func (g *GenesisBlockGenerator) createChannelConfigGroup() (*ConfigGroup, error) {
	// Create orderer configuration group
	ordererGroup, err := g.createOrdererConfigGroup()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create orderer config group")
	}

	// Create consortiums configuration group
	consortiumsGroup, err := g.createConsortiumsConfigGroup()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create consortiums config group")
	}

	// Create channel configuration group
	channelGroup := &ConfigGroup{
		Version: 0,
		Groups: map[string]*ConfigGroup{
			"Orderer":     ordererGroup,
			"Consortiums": consortiumsGroup,
		},
		Values: map[string]*ConfigValue{
			"HashingAlgorithm": {
				Version: 0,
				Value: map[string]string{
					"name": HashAlgorithmSHA256,
				},
				ModPolicy: "Admins",
			},
			"BlockDataHashingStructure": {
				Version: 0,
				Value: map[string]uint32{
					"width": 4294967295,
				},
				ModPolicy: "Admins",
			},
			"Capabilities": {
				Version:   0,
				Value:     g.config.Capabilities,
				ModPolicy: "Admins",
			},
		},
		Policies:  g.config.Policies,
		ModPolicy: "Admins",
	}

	return channelGroup, nil
}

// createOrdererConfigGroup creates the orderer configuration group
func (g *GenesisBlockGenerator) createOrdererConfigGroup() (*ConfigGroup, error) {
	// Create configuration groups for orderer organizations
	ordererOrgs := make(map[string]*ConfigGroup)
	for _, org := range g.config.OrdererOrgs {
		orgGroup, err := g.createOrgConfigGroup(org)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create orderer org %s config group", org.Name)
		}
		ordererOrgs[org.Name] = orgGroup
	}

	ordererGroup := &ConfigGroup{
		Version: 0,
		Groups:  ordererOrgs,
		Values: map[string]*ConfigValue{
			"ConsensusType": {
				Version: 0,
				Value: map[string]any{
					"type": DefaultConsensusType, // Using solo consensus for simple implementation
				},
				ModPolicy: "Admins",
			},
			"BatchSize": {
				Version:   0,
				Value:     g.config.BatchSize,
				ModPolicy: "Admins",
			},
			"BatchTimeout": {
				Version: 0,
				Value: map[string]string{
					"timeout": g.config.BatchTimeout,
				},
				ModPolicy: "Admins",
			},
			"Capabilities": {
				Version: 0,
				Value: map[string]bool{
					CapabilityV2_0: true,
				},
				ModPolicy: "Admins",
			},
		},
		Policies: map[string]*Policy{
			"Readers":         createImplicitMetaPolicy(PolicyRuleAny, "Readers"),
			"Writers":         createImplicitMetaPolicy(PolicyRuleAny, "Writers"),
			"Admins":          createImplicitMetaPolicy(PolicyRuleMajority, "Admins"),
			"BlockValidation": createImplicitMetaPolicy(PolicyRuleAny, "Writers"),
		},
		ModPolicy: "Admins",
	}

	return ordererGroup, nil
}

// createConsortiumsConfigGroup creates the consortiums configuration group
func (g *GenesisBlockGenerator) createConsortiumsConfigGroup() (*ConfigGroup, error) {
	// Create consortium configuration
	consortium := &ConfigGroup{
		Version: 0,
		Groups:  make(map[string]*ConfigGroup),
		Values: map[string]*ConfigValue{
			"ChannelCreationPolicy": {
				Version:   0,
				Value:     createImplicitMetaPolicy(PolicyRuleAny, "Admins"),
				ModPolicy: "Admins",
			},
		},
		Policies:  make(map[string]*Policy),
		ModPolicy: "Admins",
	}

	// Add peer organizations to consortium
	for _, org := range g.config.PeerOrgs {
		orgGroup, err := g.createOrgConfigGroup(org)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create peer org %s config group", org.Name)
		}
		consortium.Groups[org.Name] = orgGroup
	}

	consortiumsGroup := &ConfigGroup{
		Version: 0,
		Groups: map[string]*ConfigGroup{
			g.config.ConsortiumName: consortium,
		},
		Values:    make(map[string]*ConfigValue),
		Policies:  make(map[string]*Policy),
		ModPolicy: "Admins",
	}

	return consortiumsGroup, nil
}

// createOrgConfigGroup creates organization configuration group
func (g *GenesisBlockGenerator) createOrgConfigGroup(org *OrganizationConfig) (*ConfigGroup, error) {
	// Create MSP configuration
	mspConfig, err := g.createMSPConfig(org)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create MSP config for org %s", org.Name)
	}

	orgGroup := &ConfigGroup{
		Version: 0,
		Groups:  make(map[string]*ConfigGroup),
		Values: map[string]*ConfigValue{
			"MSP": {
				Version:   0,
				Value:     mspConfig,
				ModPolicy: "Admins",
			},
		},
		Policies:  g.createOrgPolicies(org.ID),
		ModPolicy: "Admins",
	}

	return orgGroup, nil
}

// createOrgPolicies creates standard organization policies
func (g *GenesisBlockGenerator) createOrgPolicies(mspID string) map[string]*Policy {
	createRolePolicy := func(role string) *Policy {
		return &Policy{
			Type: PolicyTypeSignature,
			Rule: &SignatureRule{
				Identities: []PolicyIdentity{
					{
						Principal: PolicyPrincipal{
							MSPIdentifier: mspID,
							Role:          role,
						},
						PrincipalClassification: "ROLE",
					},
				},
				Rule: NOutOfRule{
					NOutOf: NOutOfSpec{
						N: 1,
						Rules: []map[string]any{
							{"signed_by": 0},
						},
					},
				},
			},
		}
	}

	return map[string]*Policy{
		"Readers": createRolePolicy(RoleMember),
		"Writers": createRolePolicy(RoleMember),
		"Admins":  createRolePolicy(RoleAdmin),
	}
}

// createMSPConfig creates MSP configuration for an organization
func (g *GenesisBlockGenerator) createMSPConfig(org *OrganizationConfig) (*msp.MSPConfig, error) {
	// Load CA certificates
	caCerts, err := msp.LoadCACerts(org.MSPDir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load CA certs for org %s", org.Name)
	}

	// Load TLS CA certificates (optional)
	tlsCACerts, err := msp.LoadTLSCACerts(org.MSPDir)
	if err != nil {
		// Use regular CA certificates if TLS CA certificates are not available
		// This is acceptable for development but should be logged in production
		tlsCACerts = caCerts
	}

	// Validate that we have at least one certificate
	if len(caCerts) == 0 {
		return nil, errors.Errorf("no CA certificates found for org %s", org.Name)
	}

	mspConfig := &msp.MSPConfig{
		MSPID:        org.ID,
		RootCerts:    caCerts,
		TLSRootCerts: tlsCACerts,
		// NodeOUs: &msp.FabricNodeOUs{
		// 	Enable: true,
		// 	ClientOUIdentifier: &msp.FabricOUIdentifier{
		// 		OrganizationalUnitIdentifier: OUClient,
		// 	},
		// 	PeerOUIdentifier: &msp.FabricOUIdentifier{
		// 		OrganizationalUnitIdentifier: OUPeer,
		// 	},
		// 	AdminOUIdentifier: &msp.FabricOUIdentifier{
		// 		OrganizationalUnitIdentifier: OUAdmin,
		// 	},
		// 	OrdererOUIdentifier: &msp.FabricOUIdentifier{
		// 		OrganizationalUnitIdentifier: OUOrderer,
		// 	},
		// },
	}

	return mspConfig, nil
}

// DefaultGenesisConfig creates a default genesis configuration
func DefaultGenesisConfig() *GenesisConfig {
	return &GenesisConfig{
		NetworkName:    DefaultNetworkName,
		ConsortiumName: DefaultConsortiumName,
		OrdererOrgs: []*OrganizationConfig{
			{
				Name:     "OrdererOrg",
				ID:       "OrdererMSP",
				MSPDir:   "./ca/ca-client/orderer0/msp",
				MSPType:  MSPTypeBCCSP,
				Policies: make(map[string]*Policy),
			},
		},
		PeerOrgs: []*OrganizationConfig{
			{
				Name:     "Org1",
				ID:       "Org1MSP",
				MSPDir:   "./ca/ca-client/peer0/msp",
				MSPType:  MSPTypeBCCSP,
				Policies: make(map[string]*Policy),
			},
		},
		SystemChannel: &SystemChannelConfig{
			Name:       DefaultSystemChannel,
			Consortium: DefaultConsortiumName,
			Capabilities: map[string]bool{
				CapabilityV2_0: true,
			},
			Policies: make(map[string]*Policy),
		},
		Capabilities: map[string]bool{
			CapabilityV2_0: true,
		},
		Policies: createDefaultChannelPolicies(),
		BatchSize: &BatchSizeConfig{
			MaxMessageCount:   DefaultMaxMessageCount,
			AbsoluteMaxBytes:  DefaultAbsoluteMaxBytes,
			PreferredMaxBytes: DefaultPreferredMaxBytes,
		},
		BatchTimeout: DefaultBatchTimeout,
	}
}

// createDefaultChannelPolicies creates default channel-level policies
func createDefaultChannelPolicies() map[string]*Policy {
	return map[string]*Policy{
		"Readers": createImplicitMetaPolicy(PolicyRuleAny, "Readers"),
		"Writers": createImplicitMetaPolicy(PolicyRuleAny, "Writers"),
		"Admins":  createImplicitMetaPolicy(PolicyRuleMajority, "Admins"),
	}
}

// createImplicitMetaPolicy creates an implicit meta policy with proper typing
func createImplicitMetaPolicy(rule, subPolicy string) *Policy {
	return &Policy{
		Type: PolicyTypeImplicitMeta,
		Rule: &ImplicitMetaRule{
			Rule:      rule,
			SubPolicy: subPolicy,
		},
	}
}
