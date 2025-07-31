package config

import (
	"bufio"
	"os"
	"strings"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/pkg/errors"
)

type PeerConfig struct {
	MSPPath     string
	MSPID       string
	Address     string
	TLSEnabled  bool
	TLSCertFile string
}

type OrdererConfig struct {
	MSPPath string
	MSPID   string
	Address string
}

type AdminConfig struct {
	MSPPath string
	MSPID   string
}

type ChannelConfig struct {
	Name    string
	Profile string
}

type Config struct {
	Peer    *PeerConfig
	Orderer *OrdererConfig
	Admin   *AdminConfig
	Channel *ChannelConfig
}

func LoadPeerConfig(peerName string) (*Config, error) {

	if err := loadEnvFile(); err != nil {
		return nil, errors.Wrap(err, "failed to load .env file")
	}

	// peer 이름에서 조직과 peer 번호 추출 (예: org1peer0 -> org1, peer0)
	org, peer := parsePeerName(peerName)
	logger.Debugf("org: %s", org)
	logger.Debugf("peer: %s", peer)

	// 환경변수 prefix 생성 (예: ORG1_PEER0_)
	envPrefix := strings.ToUpper(org) + "_" + strings.ToUpper(peer) + "_"

	config := &Config{
		Peer: &PeerConfig{
			MSPPath:     getEnvOrDefault(envPrefix+"MSP_PATH", "/Users/mac/go/src/github.com/ddr4869/minifab/ca/Org1/ca-client/peer0/msp"),
			MSPID:       getEnvOrDefault(envPrefix+"MSPID", "Org1MSP"),
			Address:     getEnvOrDefault(envPrefix+"ADDRESS", "127.0.0.1:7051"),
			TLSEnabled:  getEnvBoolOrDefault("TLS_ENABLED", false),
			TLSCertFile: getEnvOrDefault("TLS_ROOTCERT_FILE", "/Users/mac/go/src/github.com/ddr4869/minifab/ca/Org1/ca-client/peer0/msp/cacerts/ca.crt"),
		},
		Orderer: &OrdererConfig{
			MSPPath: getEnvOrDefault("ORDERER_MSP_PATH", "/Users/mac/go/src/github.com/ddr4869/minifab/ca/OrdererOrg/ca-client/orderer0/msp"),
			MSPID:   getEnvOrDefault("ORDERER_MSPID", "OrdererMSP"),
			Address: getEnvOrDefault("ORDERER_ADDRESS", "127.0.0.1:7050"),
		},
		Admin: &AdminConfig{
			MSPPath: getEnvOrDefault("ADMIN_MSP_PATH", "/Users/mac/go/src/github.com/ddr4869/minifab/ca/Org1/ca-client/admin/msp"),
			MSPID:   getEnvOrDefault("ADMIN_MSPID", "Org1MSP"),
		},
		Channel: &ChannelConfig{
			Name:    getEnvOrDefault("CHANNEL_NAME", "mychannel"),
			Profile: getEnvOrDefault("PROFILE_NAME", "testchannel0"),
		},
	}

	return config, nil
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

// loadEnvFile .env 파일 로드
func loadEnvFile() error {
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

		// 빈 줄이나 주석 무시
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// KEY=VALUE 파싱
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // 잘못된 형식은 무시
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// 따옴표 제거
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

func (c *Config) GetAdminMSPPath() string {
	return c.Admin.MSPPath
}

func (c *Config) PrintConfig() {
	logger.Infof("=== Configuration ===")
	logger.Infof(" > Peer MSP Path: %s", c.Peer.MSPPath)
	logger.Infof(" > Peer MSP ID: %s", c.Peer.MSPID)
	logger.Infof(" > Peer Address: %s", c.Peer.Address)
	logger.Infof(" > TLS Enabled: %t", c.Peer.TLSEnabled)
	logger.Infof(" > Orderer MSP Path: %s", c.Orderer.MSPPath)
	logger.Infof(" > Orderer Address: %s", c.Orderer.Address)
	logger.Infof(" > Admin MSP Path: %s", c.Admin.MSPPath)
	logger.Infof(" > Channel Name: %s", c.Channel.Name)
	logger.Infof(" > Profile Name: %s", c.Channel.Profile)
}
