package configtx

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
