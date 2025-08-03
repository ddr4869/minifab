package bootstrap

import (
	"encoding/json"
	"os"

	"github.com/ddr4869/minifab/common/blockutil"
	"github.com/ddr4869/minifab/common/configtx"
	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

var (
	address      string
	mspID        string
	mspPath      string
	genesisPath  string
	configTxPath string
	profile      string
	bootstrap    bool
)

const (
	systemChannelName    = "SYSTEM_CHANNEL"
	genesisBlockJsonPath = "/Users/mac/go/src/github.com/ddr4869/minifab/nodedata/orderer0/genesis.json"
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
	bootstrapCmd.PersistentFlags().StringVar(&mspPath, "mspdir", "/Users/mac/go/src/github.com/ddr4869/minifab/ca/OrdererOrg/ca-client/orderer0", "Path to MSP directory with certificates")

	// Bootstrap command flags
	bootstrapCmd.Flags().StringVar(&genesisPath, "genesisPath", "/Users/mac/go/src/github.com/ddr4869/minifab/nodedata/orderer0/genesis.block", "Path to save/load genesis block file")
	bootstrapCmd.Flags().StringVar(&configTxPath, "configtx", "/Users/mac/go/src/github.com/ddr4869/minifab/config/configtx.yaml", "Path to configtx.yaml file")
	bootstrapCmd.Flags().StringVar(&profile, "profile", "SystemChannel", "Profile name to use for genesis block")
	bootstrapCmd.Flags().BoolVar(&bootstrap, "bootstrap", false, "Bootstrap network with genesis block")

	return bootstrapCmd
}

func runBootstrap(cmd *cobra.Command, args []string) {
	logger.Info("Starting network bootstrap process...")

	// configtx.yaml에서 제네시스 설정 생성 (profile 인자 추가)
	genesisConfig, err := CreateGenesisConfigFromConfigTx(configTxPath, profile)
	if err != nil {
		logger.Fatalf("Failed to load configtx.yaml: %v", err)
	}

	logger.Info("Successfully loaded configuration from configtx.yaml")

	// 네트워크 부트스트랩 실행
	if err := bootstrapNetwork(genesisConfig); err != nil {
		logger.Fatalf("Failed to bootstrap network: %v", err)
	}

	logger.Info("Network bootstrap completed successfully!")
	logger.Infof("Configuration loaded from: %s", configTxPath)
	logger.Info("You can now start the orderer with: ./bin/orderer")
}

// CreateGenesisConfigFromConfigTx configtx.yaml 파일에서 ConfigTx 생성
func CreateGenesisConfigFromConfigTx(configTxPath string, profile string) (*configtx.SystemChannelInfo, error) {

	ccfg, err := configtx.ConvertConfigtx(configTxPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert configtx")
	}
	// ConfigTxYAML을 ConfigTx로 변환
	genesisConfig, err := ccfg.GetSystemChannelInfo(profile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert configtx to genesis config")
	}

	logger.Infof("Successfully loaded configuration from %s with profile %s", configTxPath, profile)

	return genesisConfig, nil
}

func bootstrapNetwork(genesisConfig *configtx.SystemChannelInfo) error {

	err := generateGenesisBlock(genesisConfig)
	if err != nil {
		return errors.Wrap(err, "failed to generate genesis block")
	}

	logger.Info("Genesis block created and saved successfully")

	return nil
}

func generateGenesisBlock(genesisConfig *configtx.SystemChannelInfo) error {
	configTxData, err := json.Marshal(genesisConfig)
	if err != nil {
		return errors.Wrap(err, "failed to marshal genesis config")
	}
	msp, err := msp.LoadMSPFromFiles(mspID, mspPath)
	if err != nil {
		return errors.Wrap(err, "failed to load MSP")
	}

	genesisBlock, err := blockutil.GenerateConfigBlock(configTxData, systemChannelName, msp.GetSigningIdentity())
	if err != nil {
		return errors.Wrap(err, "failed to generate genesis block")
	}

	protoData, err := proto.Marshal(genesisBlock)
	if err != nil {
		return errors.Wrap(err, "failed to marshal genesis block")
	}

	if err := os.WriteFile(genesisPath, protoData, 0644); err != nil {
		return errors.Wrap(err, "failed to write genesis block file")
	}
	logger.Info("Genesis block created and saved at %s successfully", genesisPath)

	jsonData, err := json.MarshalIndent(genesisBlock, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal genesis block to JSON")
	}

	if err := os.WriteFile(genesisBlockJsonPath, jsonData, 0644); err != nil {
		return errors.Wrap(err, "failed to write genesis JSON file")
	}
	logger.Info("Genesis info created and saved at genesis.block successfully")

	return nil
}
