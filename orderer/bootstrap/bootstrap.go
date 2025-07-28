package bootstrap

import (
	"os"

	"github.com/ddr4869/minifab/common/configtx"
	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/orderer"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	address      string
	mspID        string
	mspPath      string
	genesisFile  string
	configTxPath string
	profile      string
	bootstrap    bool
)

func Cmd() *cobra.Command {

	bootstrapCmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap the blockchain network with genesis block",
		Long: `Bootstrap the blockchain network by generating and initializing the genesis block.
This command should be run once when setting up a new network.`,
		Run: runBootstrap,
	}

	bootstrapCmd.PersistentFlags().StringVar(&address, "address", "0.0.0.0:7050", "Orderer server address")
	bootstrapCmd.PersistentFlags().StringVar(&mspID, "mspid", "OrdererMSP", "MSP ID for orderer")
	bootstrapCmd.PersistentFlags().StringVar(&mspPath, "mspdir", "./ca/ca-client/orderer0/msp", "Path to MSP directory with certificates")

	// Bootstrap command flags
	bootstrapCmd.Flags().StringVar(&genesisFile, "genesisFile", "./config/genesis.json", "Path to save/load genesis block file")
	bootstrapCmd.Flags().StringVar(&configTxPath, "configtx", "./config/configtx.yaml", "Path to configtx.yaml file")
	bootstrapCmd.Flags().StringVar(&profile, "profile", "SystemChannel", "Profile name to use for genesis block")
	bootstrapCmd.Flags().BoolVar(&bootstrap, "bootstrap", false, "Bootstrap network with genesis block")

	return bootstrapCmd
}

func runBootstrap(cmd *cobra.Command, args []string) {
	logger.Info("Starting network bootstrap process...")

	// Orderer 인스턴스 생성 (MSP 파일 사용)
	o, err := orderer.NewOrderer(mspID, mspPath)
	if err != nil {
		logger.Fatalf("Failed to create orderer: %v", err)
	}

	// configtx.yaml에서 제네시스 설정 생성 (profile 인자 추가)
	genesisConfig, err := CreateGenesisConfigFromConfigTx(configTxPath, profile)
	if err != nil {
		logger.Fatalf("Failed to load configtx.yaml: %v", err)
	}

	logger.Info("Successfully loaded configuration from configtx.yaml")

	// 네트워크 부트스트랩 실행
	if err := o.BootstrapNetwork(genesisConfig); err != nil {
		logger.Fatalf("Failed to bootstrap network: %v", err)
	}

	logger.Info("Network bootstrap completed successfully!")
	logger.Infof("Configuration loaded from: %s", configTxPath)
	logger.Info("You can now start the orderer with: ./bin/orderer")
}

// CreateGenesisConfigFromConfigTx configtx.yaml 파일에서 ConfigTx 생성
func CreateGenesisConfigFromConfigTx(configTxPath string, profile string) (*configtx.SystemChannelInfo, error) {
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
	var configTx configtx.ConfigTx
	if err := yaml.Unmarshal(data, &configTx); err != nil {
		return nil, errors.Wrap(err, "failed to parse configtx YAML")
	}

	// ConfigTxYAML을 ConfigTx로 변환
	genesisConfig, err := configTx.GetSystemChannelInfo(profile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert configtx to genesis config")
	}

	logger.Infof("Successfully loaded configuration from %s with profile %s", configTxPath, profile)

	return genesisConfig, nil
}
