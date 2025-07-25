package configtx

type ConfigTx struct {
	Organizations []Organization     `yaml:"Organizations"`
	Orderer       Orderer            `yaml:"Orderer"`
	Channel       Channel            `yaml:"Channel"`
	Profiles      map[string]Profile `yaml:"Profiles"`
}

type Organization struct {
	Name             string       `yaml:"Name"`
	ID               string       `yaml:"ID"`
	MSPDir           string       `yaml:"MSPDir"`
	OrdererEndpoints []string     `yaml:"OrdererEndpoints,omitempty"`
	AnchorPeers      []AnchorPeer `yaml:"AnchorPeers,omitempty"`
}

type AnchorPeer struct {
	Host string `yaml:"Host"`
	Port int    `yaml:"Port"`
}

type Orderer struct {
	Type          string    `yaml:"Type"`
	BatchTimeout  string    `yaml:"BatchTimeout"`
	BatchSize     BatchSize `yaml:"BatchSize"`
	Organizations []string  `yaml:"Organizations"`
}

type BatchSize struct {
	MaxMessageCount   int    `yaml:"MaxMessageCount"`
	AbsoluteMaxBytes  string `yaml:"AbsoluteMaxBytes"`
	PreferredMaxBytes string `yaml:"PreferredMaxBytes"`
}

type Channel struct {
	Policies string `yaml:"Policies"` // any, all, majority
}

type Profile struct {
	Orderer       *Orderer     `yaml:"Orderer,omitempty"`
	Application   *Application `yaml:"Application,omitempty"`
	Organizations []string     `yaml:"Organizations"`
	Consortium    string       `yaml:"Consortium,omitempty"`
}

type Application struct {
	Organizations []string `yaml:"Organizations"`
}
