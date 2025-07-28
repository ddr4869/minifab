package configtx

// type Organization struct {
// 	Name             string       `yaml:"Name"`
// 	ID               string       `yaml:"ID"`
// 	MSPDir           string       `yaml:"MSPDir"`
// 	OrdererEndpoints []string     `yaml:"OrdererEndpoints,omitempty"`
// 	AnchorPeers      []AnchorPeer `yaml:"AnchorPeers,omitempty"`
// }

// type AnchorPeer struct {
// 	Host string `yaml:"Host"`
// 	Port int    `yaml:"Port"`
// }

// type Orderer struct {
// 	Type          string    `yaml:"Type,omitempty"`
// 	BatchTimeout  string    `yaml:"BatchTimeout"`
// 	BatchSize     BatchSize `yaml:"BatchSize"`
// 	Organizations []string  `yaml:"Organizations"` // YAML 참조를 위한 문자열 배열
// }

// type BatchSize struct {
// 	MaxMessageCount   int    `yaml:"MaxMessageCount"`
// 	AbsoluteMaxBytes  string `yaml:"AbsoluteMaxBytes"`
// 	PreferredMaxBytes string `yaml:"PreferredMaxBytes"`
// }

// type Channel struct {
// 	Policies string `yaml:"Policies"` // any, all, majority
// }

// type Application struct {
// 	Organizations []string `yaml:"Organizations"` // YAML 참조를 위한 문자열 배열
// }
