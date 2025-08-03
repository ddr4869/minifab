package config

import (
	"bufio"
	"os"
	"strings"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
	"github.com/pkg/errors"
)

type PeerCfg struct {
	MSPPath        string
	MSPID          string
	MSP            msp.MSP
	Address        string
	FilesystemPath string
	TLSEnabled     bool
	TLSCertFile    string
}

type OrdererCfg struct {
	MSPPath        string
	MSPID          string
	MSP            msp.MSP
	Address        string
	FilesystemPath string
	GenesisPath    string
}

type ClientCfg struct {
	MSPPath string
	MSPID   string
	MSP     msp.MSP
}

type ChannelCfg struct {
	Name    string
	Profile string
	// TODO : ChannelMSP
}

type Config struct {
	Peer    *PeerCfg
	Orderer *OrdererCfg
	Client  *ClientCfg
	Channel *ChannelCfg
}

func LoadPeerConfig(peerName string) (*Config, error) {

	err := LoadEnvFile()
	if err != nil {
		logger.Errorf("Failed to load env file: %v", err)
		return nil, err
	}
	org, peer := parsePeerName(peerName)
	logger.Debugf("org, peer: %s, %s", org, peer)

	orgPrefix := strings.ToUpper(org) + "_"
	peerPrefix := strings.ToUpper(peer) + "_"
	envPrefix := orgPrefix + peerPrefix

	config := &Config{
		Peer: &PeerCfg{
			MSPPath:        getEnvOrDefault(envPrefix+"MSP_PATH", "/Users/mac/go/src/github.com/ddr4869/minifab/ca/Org1/ca-client/peer0"),
			MSPID:          getEnvOrDefault(envPrefix+"MSPID", "Org1MSP"),
			Address:        getEnvOrDefault(envPrefix+"ADDRESS", "127.0.0.1:7051"),
			FilesystemPath: getEnvOrDefault(envPrefix+"FILESYSTEM_PATH", "/Users/mac/go/src/github.com/ddr4869/minifab/nodedata/org1peer0"),
			TLSEnabled:     getEnvBoolOrDefault("TLS_ENABLED", false),
			TLSCertFile:    getEnvOrDefault("TLS_ROOTCERT_FILE", "/Users/mac/go/src/github.com/ddr4869/minifab/ca/Org1/ca-client/peer0/cacerts/ca.crt"),
		},
		Orderer: &OrdererCfg{
			MSPPath:        getEnvOrDefault("ORDERER_MSP_PATH", "/Users/mac/go/src/github.com/ddr4869/minifab/ca/OrdererOrg/ca-client/orderer0"),
			MSPID:          getEnvOrDefault("ORDERER_MSPID", "OrdererMSP"),
			Address:        getEnvOrDefault("ORDERER_ADDRESS", "127.0.0.1:7050"),
			FilesystemPath: getEnvOrDefault("ORDERER_FILESYSTEM_PATH", "/Users/mac/go/src/github.com/ddr4869/minifab/nodedata/orderer0"),
			GenesisPath:    getEnvOrDefault("GENESIS_PATH", "/Users/mac/go/src/github.com/ddr4869/minifab/nodedata/orderer0/genesis.block"),
		},
		Client: &ClientCfg{
			MSPPath: getEnvOrDefault(orgPrefix+"CLIENT_MSP_PATH", "/Users/mac/go/src/github.com/ddr4869/minifab/ca/Org1/ca-client/admin"),
			MSPID:   getEnvOrDefault(orgPrefix+"CLIENT_MSPID", "Org1MSP"),
		},
		Channel: &ChannelCfg{
			Name:    getEnvOrDefault("CHANNEL_NAME", "mychannel"),
			Profile: getEnvOrDefault("PROFILE_NAME", "testchannel0"),
		},
	}

	return config, nil
}

func LoadOrdererConfig(ordererId string) (*OrdererCfg, error) {

	err := LoadEnvFile()
	if err != nil {
		logger.Errorf("Failed to load env file: %v", err)
		return nil, err
	}
	ordererCfg := &OrdererCfg{
		MSPPath:        getEnvOrDefault("ORDERER_MSP_PATH", "/Users/mac/go/src/github.com/ddr4869/minifab/ca/OrdererOrg/ca-client/orderer0"),
		MSPID:          getEnvOrDefault("ORDERER_MSPID", "OrdererMSP"),
		Address:        getEnvOrDefault("ORDERER_ADDRESS", "127.0.0.1:7050"),
		GenesisPath:    getEnvOrDefault("GENESIS_PATH", "/Users/mac/go/src/github.com/ddr4869/minifab/nodedata/orderer0/genesis.block"),
		FilesystemPath: getEnvOrDefault("ORDERER_FILESYSTEM_PATH", "/Users/mac/go/src/github.com/ddr4869/minifab/nodedata/orderer0"),
	}

	return ordererCfg, nil
}

// parsePeerName peer 이름을 조직과 peer로 분리
func parsePeerName(peerName string) (org, peer string) {
	// org1peer0 -> org1, peer0
	// org2peer1 -> org2, peer1

	if strings.HasSuffix(peerName, "peer") {
		// org1peer -> org1, peer
		org = strings.TrimSuffix(peerName, "peer")
		peer = "peer"
	} else {
		// org1peer0 -> org1, peer0
		// org2peer1 -> org2, peer1
		for i := len(peerName) - 1; i >= 0; i-- {
			if peerName[i] >= '0' && peerName[i] <= '9' {
				continue
			}
			if strings.HasPrefix(peerName[i:], "peer") {
				org = peerName[:i]
				peer = peerName[i:]
				break
			}
		}
	}

	if org == "" {
		org = "org1"
	}
	if peer == "" {
		peer = "peer0"
	}
	return org, peer
}

func LoadEnvFile() error {
	possiblePaths := []string{
		"config/.env",
		".env",
		"../config/.env",
		"../../config/.env",
	}

	var envPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			envPath = path
			break
		}
	}

	if envPath == "" {
		return errors.New("no .env file found")
	}

	file, err := os.Open(envPath)
	if err != nil {
		return errors.Wrapf(err, "failed to open .env file: %s", envPath)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"') {
			value = value[1 : len(value)-1]
		}

		os.Setenv(key, value)
	}

	return scanner.Err()
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true"
	}
	return defaultValue
}

func (c *Config) GetPeerMSPPath() string {
	return c.Peer.MSPPath
}

func (c *Config) GetOrdererMSPPath() string {
	return c.Orderer.MSPPath
}

func (c *Config) GetClientMSPPath() string {
	return c.Client.MSPPath
}

func (c *Config) PrintConfig() {
	logger.Infof("=== Configuration ===")
	logger.Infof(" > Peer MSP Path: %s", c.Peer.MSPPath)
	logger.Infof(" > Peer MSP ID: %s", c.Peer.MSPID)
	logger.Infof(" > Peer Address: %s", c.Peer.Address)
	logger.Infof(" > TLS Enabled: %t", c.Peer.TLSEnabled)
	logger.Infof(" > Orderer MSP Path: %s", c.Orderer.MSPPath)
	logger.Infof(" > Orderer Address: %s", c.Orderer.Address)
	logger.Infof(" > Client MSP Path: %s", c.Client.MSPPath)
	logger.Infof(" > Channel Name: %s", c.Channel.Name)
	logger.Infof(" > Profile Name: %s", c.Channel.Profile)
}
