package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

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

type OrdererConfig struct {
	BatchTimeout string    `yaml:"BatchTimeout"`
	BatchSize    BatchSize `yaml:"BatchSize"`
}

type BatchSize struct {
	MaxMessageCount   int    `yaml:"MaxMessageCount"`
	AbsoluteMaxBytes  string `yaml:"AbsoluteMaxBytes"`
	PreferredMaxBytes string `yaml:"PreferredMaxBytes"`
}

type ChannelConfig struct {
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

type SystemChannelProfile struct {
	Orderer     SystemChannelConfig `yaml:"Orderer"`
	Consortiums []Organization      `yaml:"Consortiums"`
}

type AppChannelProfile struct {
	Application ChannelConfig `yaml:"Application"`
}

type Profiles struct {
	SystemChannel SystemChannelProfile `yaml:"SystemChannel"`
	AppChannel    AppChannelProfile    `yaml:"AppChannel"`
}

type Config struct {
	Organizations []Organization         `yaml:"Organizations"`
	Orderer       OrdererConfig          `yaml:"Orderer"`
	Channel       ChannelConfig          `yaml:"Channel"`
	Profiles      map[string]interface{} `yaml:"Profiles"`
}

func (c *Config) GetSystemChannelProfile(name string) (*SystemChannelProfile, error) {
	profileData, exists := c.Profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}

	yamlData, err := yaml.Marshal(profileData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal profile data: %v", err)
	}

	var systemProfile SystemChannelProfile
	if err := yaml.Unmarshal(yamlData, &systemProfile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal as SystemChannelProfile: %v", err)
	}
	return &systemProfile, nil
}

func (c *Config) GetAppChannelProfile(name string) (*AppChannelProfile, error) {
	profileData, exists := c.Profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile '%s' not found", name)
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

func main() {
	data, err := os.ReadFile("configtx.yaml")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	var configtx Config
	if err := yaml.Unmarshal(data, &configtx); err != nil {
		fmt.Println("Error unmarshalling data:", err)
		return
	}

	systemProfile, err := configtx.GetSystemChannelProfile("SystemChannel2")
	if err != nil {
		fmt.Println("Error getting system profile:", err)
		return
	}
	fmt.Println(systemProfile.Orderer.Organization.Name)

}
