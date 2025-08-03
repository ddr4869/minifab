package configtx

import (
	"fmt"
	"os"

	"github.com/ddr4869/minifab/common/cert"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Organization struct {
	Name             string       `yaml:"Name"`
	ID               string       `yaml:"ID"`
	MSPDir           string       `yaml:"MSPDir"`
	MSPCaCert        []byte       `yaml:"-"`
	OrdererEndpoints []string     `yaml:"OrdererEndpoints,omitempty"`
	AnchorPeers      []AnchorPeer `yaml:"AnchorPeers,omitempty"`
}

type AnchorPeer struct {
	Host string `yaml:"Host"`
	Port int    `yaml:"Port"`
}

type OrdererConfig struct {
	BatchTimeout string    `yaml:"BatchTimeout"`
	BatchSize    BatchSize `yaml:"BatchSize"`
}

type BatchSize struct {
	MaxMessageCount   int    `yaml:"MaxMessageCount"`
	AbsoluteMaxBytes  string `yaml:"AbsoluteMaxBytes"`
	PreferredMaxBytes string `yaml:"PreferredMaxBytes"`
}

type AppChannelConfig struct {
	Policies      interface{}    `yaml:"Policies"` // all 등 단순 문자열일 수도, 정책구조일 수도 있음
	Organizations []Organization `yaml:"Organizations"`
}

type Consortium struct {
	Organizations []Organization `yaml:"Organizations"`
}

type SystemChannelConfig struct {
	BatchTimeout string       `yaml:"BatchTimeout"`
	BatchSize    BatchSize    `yaml:"BatchSize"`
	Organization Organization `yaml:"Organization"` // YAML 참조를 위한 문자열
}

type SystemChannelInfo struct {
	Orderer     SystemChannelConfig `yaml:"Orderer"`
	Consortiums []Organization      `yaml:"Consortiums"`
}

type AppChannelProfile struct {
	Application AppChannelConfig `yaml:"Application"`
}

type Profiles struct {
	SystemChannel SystemChannelInfo `yaml:"SystemChannel"`
	AppChannel    AppChannelProfile `yaml:"AppChannel"`
}

type ConfigTx struct {
	Organizations []Organization         `yaml:"Organizations"`
	Orderer       OrdererConfig          `yaml:"Orderer"`
	Channel       AppChannelConfig       `yaml:"Channel"`
	Profiles      map[string]interface{} `yaml:"Profiles"`
}

type ChannelConfig struct {
	CC  *AppChannelConfig
	SCC *SystemChannelInfo
}

func (c *ConfigTx) GetSystemChannelInfo(name string) (*SystemChannelInfo, error) {
	profileData, exists := c.Profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}

	yamlData, err := yaml.Marshal(profileData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal profile data: %v", err)
	}

	var systemProfile SystemChannelInfo
	if err := yaml.Unmarshal(yamlData, &systemProfile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal as SystemChannelInfo: %v", err)
	}

	for i, org := range systemProfile.Consortiums {
		cert, err := cert.LoadCaCertFromDir(org.MSPDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load certificate")
		}
		systemProfile.Consortiums[i].MSPCaCert = cert.Raw
	}
	return &systemProfile, nil
}

func (c *ConfigTx) GetAppChannelProfile(profileName string) (*AppChannelProfile, error) {
	profileData, exists := c.Profiles[profileName]
	if !exists {
		return nil, fmt.Errorf("profile '%s' not found", profileName)
	}

	yamlData, err := yaml.Marshal(profileData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal profile data: %v", err)
	}

	var appChannelProfile AppChannelProfile
	if err := yaml.Unmarshal(yamlData, &appChannelProfile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal as AppChannelProfile: %v", err)
	}
	return &appChannelProfile, nil
}

func ConvertConfigtx(configTxPath string) (*ConfigTx, error) {
	if configTxPath == "" {
		return nil, errors.Errorf("configtx path cannot be empty")
	}

	// configtx.yaml 파일 존재 확인
	if _, err := os.Stat(configTxPath); os.IsNotExist(err) {
		return nil, errors.Errorf("configtx file does not exist: %s", configTxPath)
	}

	// configtx.yaml 파일 읽기
	data, err := os.ReadFile(configTxPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read configtx file")
	}

	// YAML 파싱
	var configTx ConfigTx
	if err := yaml.Unmarshal(data, &configTx); err != nil {
		return nil, errors.Wrap(err, "failed to parse configtx YAML")
	}

	for i, org := range configTx.Organizations {
		cert, err := cert.LoadCaCertFromDir(org.MSPDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load certificate")
		}
		configTx.Organizations[i].MSPCaCert = cert.Raw
	}

	return &configTx, nil
}

// parseBatchSizeBytes 크기 문자열을 바이트 수로 변환 ("128 MB" -> 134217728)
func ParseBatchSizeBytes(sizeStr string) (uint32, error) {
	if sizeStr == "" {
		return 0, errors.New("size string cannot be empty")
	}

	var value uint32
	var unit string

	// 숫자와 단위 분리
	n, err := fmt.Sscanf(sizeStr, "%d %s", &value, &unit)
	if err != nil || n != 2 {
		// 단위 없이 숫자만 있는 경우
		if n, err := fmt.Sscanf(sizeStr, "%d", &value); err != nil || n != 1 {
			return 0, errors.Errorf("failed to parse size: %s", sizeStr)
		}
		return value, nil
	}

	// 단위에 따른 배수 적용
	switch unit {
	case "KB":
		return value * 1024, nil
	case "MB":
		return value * 1024 * 1024, nil
	case "GB":
		return value * 1024 * 1024 * 1024, nil
	default:
		return 0, errors.Errorf("unsupported size unit: %s", unit)
	}
}
