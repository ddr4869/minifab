package configtx

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// ParseConfigTx parses a configtx.yaml file
func ParseConfigTx(filePath string) (*ConfigTx, error) {
	if filePath == "" {
		return nil, errors.New("file path cannot be empty")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read configtx file")
	}

	var configTx ConfigTx
	if err := yaml.Unmarshal(data, &configTx); err != nil {
		return nil, errors.Wrap(err, "failed to parse configtx YAML")
	}

	return &configTx, nil
}

// // ConvertToGenesisConfig converts parsed configtx to GenesisConfig format
// func (c *ConfigTx) ConvertToGenesisConfig() (*GenesisConfig, error) {
// 	// Get organizations
// 	ordererOrgs := c.GetOrdererOrganizations()
// 	peerOrgs := c.GetPeerOrganizations()

// 	if len(ordererOrgs) == 0 {
// 		return nil, errors.New("no orderer organizations found in configtx")
// 	}

// 	// Convert orderer organizations
// 	var genesisOrdererOrgs []*OrganizationConfig
// 	for _, org := range ordererOrgs {
// 		genesisOrg := &OrganizationConfig{
// 			Name:     org.Name,
// 			ID:       org.ID,
// 			MSPDir:   org.MSPDir,
// 			MSPType:  "bccsp", // Default MSP type
// 			Policies: convertPolicies(org.Policies),
// 		}
// 		genesisOrdererOrgs = append(genesisOrdererOrgs, genesisOrg)
// 	}

// 	// Convert peer organizations
// 	var genesisPeerOrgs []*OrganizationConfig
// 	for _, org := range peerOrgs {
// 		genesisOrg := &OrganizationConfig{
// 			Name:     org.Name,
// 			ID:       org.ID,
// 			MSPDir:   org.MSPDir,
// 			MSPType:  "bccsp", // Default MSP type
// 			Policies: convertPolicies(org.Policies),
// 		}
// 		genesisPeerOrgs = append(genesisPeerOrgs, genesisOrg)
// 	}

// 	// Parse batch size
// 	batchSize, err := c.parseBatchSizeConfig()
// 	if err != nil {
// 		return nil, errors.Wrap(err, "failed to parse batch size")
// 	}

// 	// Create genesis config
// 	genesisConfig := &GenesisConfig{
// 		NetworkName:    "CustomFabricNetwork", // Default network name
// 		ConsortiumName: "SampleConsortium",    // Default consortium name
// 		OrdererOrgs:    genesisOrdererOrgs,
// 		PeerOrgs:       genesisPeerOrgs,
// 		SystemChannel: &SystemChannelConfig{
// 			Name:       "system-channel", // Default system channel name
// 			Consortium: "SampleConsortium",
// 			Policies:   convertPolicies(c.Channel.Policies),
// 		},
// 		Policies:     convertPolicies(c.Channel.Policies),
// 		BatchSize:    batchSize,
// 		BatchTimeout: c.Orderer.BatchTimeout,
// 	}

// 	return genesisConfig, nil
// }

// ParseBatchSizeBytes converts batch size string to bytes
func ParseBatchSizeBytes(sizeStr string) (uint32, error) {
	if sizeStr == "" {
		return 0, errors.New("size string cannot be empty")
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
		return 0, errors.Wrap(err, "failed to parse size value")
	}

	return numValue * multiplier, nil
}

// parseBatchSizeConfig parses batch size configuration from configtx
func (c *ConfigTx) parseBatchSizeConfig() (*BatchSizeConfig, error) {
	absoluteMaxBytes, err := ParseBatchSizeBytes(c.Orderer.BatchSize.AbsoluteMaxBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse absolute max bytes")
	}

	preferredMaxBytes, err := ParseBatchSizeBytes(c.Orderer.BatchSize.PreferredMaxBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse preferred max bytes")
	}

	return &BatchSizeConfig{
		MaxMessageCount:   uint32(c.Orderer.BatchSize.MaxMessageCount),
		AbsoluteMaxBytes:  absoluteMaxBytes,
		PreferredMaxBytes: preferredMaxBytes,
	}, nil
}
