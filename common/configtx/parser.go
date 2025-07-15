package configtx

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ConfigTx represents the structure of configtx.yaml
type ConfigTx struct {
	Organizations []Organization `yaml:"Organizations"`
	Application   Application    `yaml:"Application"`
	Orderer       Orderer        `yaml:"Orderer"`
	Channel       Channel        `yaml:"Channel"`
}

// Organization represents an organization configuration
type Organization struct {
	Name             string            `yaml:"Name"`
	ID               string            `yaml:"ID"`
	MSPDir           string            `yaml:"MSPDir"`
	Policies         map[string]Policy `yaml:"Policies"`
	OrdererEndpoints []string          `yaml:"OrdererEndpoints,omitempty"`
	AnchorPeers      []AnchorPeer      `yaml:"AnchorPeers,omitempty"`
}

// AnchorPeer represents an anchor peer configuration
type AnchorPeer struct {
	Host string `yaml:"Host"`
	Port int    `yaml:"Port"`
}

// Policy represents a policy configuration
type Policy struct {
	Type string `yaml:"Type"`
	Rule string `yaml:"Rule"`
}

// Application represents application configuration
type Application struct {
	Organizations []interface{}     `yaml:"Organizations"`
	Policies      map[string]Policy `yaml:"Policies"`
}

// Orderer represents orderer configuration
type Orderer struct {
	OrdererType   string            `yaml:"OrdererType"`
	BatchTimeout  string            `yaml:"BatchTimeout"`
	BatchSize     BatchSize         `yaml:"BatchSize"`
	Organizations []interface{}     `yaml:"Organizations"`
	Policies      map[string]Policy `yaml:"Policies"`
}

// BatchSize represents batch size configuration
type BatchSize struct {
	MaxMessageCount   int    `yaml:"MaxMessageCount"`
	AbsoluteMaxBytes  string `yaml:"AbsoluteMaxBytes"`
	PreferredMaxBytes string `yaml:"PreferredMaxBytes"`
}

// Channel represents channel configuration
type Channel struct {
	Policies map[string]Policy `yaml:"Policies"`
}

// Consortiums represents consortium configurations
type Consortiums struct {
	SampleConsortium Consortium `yaml:"SampleConsortium"`
}

// Consortium represents a consortium configuration
type Consortium struct {
	Organizations []interface{} `yaml:"Organizations"`
}

// ParseConfigTx parses a configtx.yaml file
func ParseConfigTx(filePath string) (*ConfigTx, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configtx file: %w", err)
	}

	var configTx ConfigTx
	if err := yaml.Unmarshal(data, &configTx); err != nil {
		return nil, fmt.Errorf("failed to parse configtx YAML: %w", err)
	}

	return &configTx, nil
}

// GetOrdererOrganizations returns orderer organizations from the parsed config
func (c *ConfigTx) GetOrdererOrganizations() []Organization {
	var ordererOrgs []Organization

	// Find organizations that have OrdererEndpoints (indicating they are orderer orgs)
	for _, org := range c.Organizations {
		if len(org.OrdererEndpoints) > 0 {
			ordererOrgs = append(ordererOrgs, org)
		}
	}

	return ordererOrgs
}

// GetPeerOrganizations returns peer organizations from the parsed config
func (c *ConfigTx) GetPeerOrganizations() []Organization {
	var peerOrgs []Organization

	// Find organizations that have AnchorPeers (indicating they are peer orgs)
	for _, org := range c.Organizations {
		if len(org.AnchorPeers) > 0 {
			peerOrgs = append(peerOrgs, org)
		}
	}

	return peerOrgs
}

// ParseBatchSizeBytes converts batch size string to bytes
func ParseBatchSizeBytes(sizeStr string) (uint32, error) {
	if sizeStr == "" {
		return 0, fmt.Errorf("size string cannot be empty")
	}

	// Handle common size suffixes
	multiplier := uint32(1)
	size := sizeStr

	if len(sizeStr) >= 2 {
		suffix := sizeStr[len(sizeStr)-2:]
		switch suffix {
		case "KB":
			multiplier = 1024
			size = sizeStr[:len(sizeStr)-2]
		case "MB":
			multiplier = 1024 * 1024
			size = sizeStr[:len(sizeStr)-2]
		case "GB":
			multiplier = 1024 * 1024 * 1024
			size = sizeStr[:len(sizeStr)-2]
		}
	}

	// Try to parse the numeric part
	var numValue uint32
	if _, err := fmt.Sscanf(size, "%d", &numValue); err != nil {
		return 0, fmt.Errorf("failed to parse size value: %w", err)
	}

	return numValue * multiplier, nil
}

// ConvertToGenesisConfig converts parsed configtx to GenesisConfig format
func (c *ConfigTx) ConvertToGenesisConfig() (*GenesisConfig, error) {
	// Get organizations
	ordererOrgs := c.GetOrdererOrganizations()
	peerOrgs := c.GetPeerOrganizations()

	if len(ordererOrgs) == 0 {
		return nil, fmt.Errorf("no orderer organizations found in configtx")
	}

	// Convert orderer organizations
	var genesisOrdererOrgs []*OrganizationConfig
	for _, org := range ordererOrgs {
		genesisOrg := &OrganizationConfig{
			Name:     org.Name,
			ID:       org.ID,
			MSPDir:   org.MSPDir,
			MSPType:  "bccsp", // Default MSP type
			Policies: convertPolicies(org.Policies),
		}
		genesisOrdererOrgs = append(genesisOrdererOrgs, genesisOrg)
	}

	// Convert peer organizations
	var genesisPeerOrgs []*OrganizationConfig
	for _, org := range peerOrgs {
		genesisOrg := &OrganizationConfig{
			Name:     org.Name,
			ID:       org.ID,
			MSPDir:   org.MSPDir,
			MSPType:  "bccsp", // Default MSP type
			Policies: convertPolicies(org.Policies),
		}
		genesisPeerOrgs = append(genesisPeerOrgs, genesisOrg)
	}

	// Parse batch size
	batchSize, err := c.parseBatchSizeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to parse batch size: %w", err)
	}

	// Create genesis config
	genesisConfig := &GenesisConfig{
		NetworkName:    "CustomFabricNetwork", // Default network name
		ConsortiumName: "SampleConsortium",    // Default consortium name
		OrdererOrgs:    genesisOrdererOrgs,
		PeerOrgs:       genesisPeerOrgs,
		SystemChannel: &SystemChannelConfig{
			Name:       "system-channel", // Default system channel name
			Consortium: "SampleConsortium",
			Policies:   convertPolicies(c.Channel.Policies),
		},
		Policies:     convertPolicies(c.Channel.Policies),
		BatchSize:    batchSize,
		BatchTimeout: c.Orderer.BatchTimeout,
	}

	return genesisConfig, nil
}

// parseBatchSizeConfig parses batch size configuration from configtx
func (c *ConfigTx) parseBatchSizeConfig() (*BatchSizeConfig, error) {
	absoluteMaxBytes, err := ParseBatchSizeBytes(c.Orderer.BatchSize.AbsoluteMaxBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse absolute max bytes: %w", err)
	}

	preferredMaxBytes, err := ParseBatchSizeBytes(c.Orderer.BatchSize.PreferredMaxBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse preferred max bytes: %w", err)
	}

	return &BatchSizeConfig{
		MaxMessageCount:   uint32(c.Orderer.BatchSize.MaxMessageCount),
		AbsoluteMaxBytes:  absoluteMaxBytes,
		PreferredMaxBytes: preferredMaxBytes,
	}, nil
}

// convertPolicies converts configtx policies to genesis config policies
func convertPolicies(policies map[string]Policy) map[string]*PolicyConfig {
	if policies == nil {
		return make(map[string]*PolicyConfig)
	}

	converted := make(map[string]*PolicyConfig)
	for name, policy := range policies {
		converted[name] = &PolicyConfig{
			Type: policy.Type,
			Rule: policy.Rule,
		}
	}
	return converted
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

// OrganizationConfig represents organization configuration
type OrganizationConfig struct {
	Name     string                   `json:"name"`
	ID       string                   `json:"id"`
	MSPDir   string                   `json:"msp_dir"`
	MSPType  string                   `json:"msp_type"`
	Policies map[string]*PolicyConfig `json:"policies"`
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
