package server

import (
	"os"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/orderer"
	"github.com/ddr4869/minifab/orderer/bootstrap"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	ordererId   string
	address     string
	mspID       string
	mspPath     string
	genesisFile string
	profile     string
)

// rootCmd는 orderer의 루트 명령어를 나타냅니다
var RootCmd = &cobra.Command{
	Use:   "orderer",
	Short: "Custom Fabric Orderer Node",
	Long: `Custom Fabric Orderer Node - 블록체인 네트워크의 트랜잭션 순서를 관리하고
블록을 생성하는 orderer 노드입니다.`,
	Run: runOrderer,
}

func init() {
	// Add subcommands
	RootCmd.Flags().StringVar(&ordererId, "ordererId", "orderer0", "Orderer ID")
	RootCmd.Flags().StringVar(&address, "address", "0.0.0.0:7050", "Orderer server address")
	RootCmd.Flags().StringVar(&mspID, "mspid", "OrdererMSP", "MSP ID for orderer")
	RootCmd.Flags().StringVar(&mspPath, "mspdir", "/Users/mac/go/src/github.com/ddr4869/minifab/ca/OrdererOrg/ca-client/orderer0", "Path to MSP directory with certificates")
	RootCmd.Flags().StringVar(&genesisFile, "genesisFile", "/Users/mac/go/src/github.com/ddr4869/minifab/blocks/genesis.json", "Path to genesis block file")
	RootCmd.Flags().StringVar(&profile, "profile", "SystemChannel", "Profile name to use for genesis block")

	RootCmd.AddCommand(bootstrap.Cmd())
}

func runOrderer(cmd *cobra.Command, args []string) {
	logger.Info("Starting orderer process...")

	// Validate input parameters
	if err := validateOrdererParams(); err != nil {
		logger.Fatalf("Invalid parameters: %v", err)
	}

	// Create orderer instance with MSP files
	node, err := orderer.NewOrderer(ordererId, mspID, mspPath, address, genesisFile)
	if err != nil {
		logger.Fatalf("Failed to create orderer: %v", err)
	}

	logger.Infof("Starting orderer server on %s with MSP ID: %s", address, mspID)
	if err := node.Start(address); err != nil {
		logger.Fatalf("Failed to start orderer server: %v", err)
	}
}

// validateOrdererParams validates orderer startup parameters
func validateOrdererParams() error {
	if address == "" {
		return errors.New("orderer address cannot be empty")
	}

	if mspID == "" {
		return errors.New("MSP ID cannot be empty")
	}

	if mspPath == "" {
		return errors.New("MSP directory path cannot be empty")
	}

	// Check if MSP directory exists
	if _, err := os.Stat(mspPath); os.IsNotExist(err) {
		return errors.Errorf("MSP directory does not exist: %s", mspPath)
	}

	return nil
}
