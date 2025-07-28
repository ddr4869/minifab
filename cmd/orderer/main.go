package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/orderer"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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

// rootCmd는 orderer의 루트 명령어를 나타냅니다
var rootCmd = &cobra.Command{
	Use:   "orderer",
	Short: "Custom Fabric Orderer Node",
	Long: `Custom Fabric Orderer Node - 블록체인 네트워크의 트랜잭션 순서를 관리하고 
블록을 생성하는 orderer 노드입니다.`,
	Run: runOrderer,
}

// bootstrapCmd는 네트워크 부트스트랩 명령어를 나타냅니다
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap the blockchain network with genesis block",
	Long: `Bootstrap the blockchain network by generating and initializing the genesis block.
This command should be run once when setting up a new network.`,
	Run: runBootstrap,
}

func init() {
	// Initialize logger with development config for CLI
	if err := logger.InitializeDevelopment(); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	// Flag definitions with better defaults
	rootCmd.PersistentFlags().StringVar(&address, "address", "0.0.0.0:7050", "Orderer server address")
	rootCmd.PersistentFlags().StringVar(&mspID, "mspid", "OrdererMSP", "MSP ID for orderer")
	rootCmd.PersistentFlags().StringVar(&mspPath, "mspdir", "./ca/ca-client/orderer0/msp", "Path to MSP directory with certificates")

	// Bootstrap command flags
	bootstrapCmd.Flags().StringVar(&genesisFile, "genesis-file", "./genesis.json", "Path to save/load genesis block file")
	bootstrapCmd.Flags().StringVar(&configTxPath, "configtx", "./config/configtx.yaml", "Path to configtx.yaml file")
	bootstrapCmd.Flags().StringVar(&profile, "profile", "SystemChannel", "Profile name to use for genesis block")
	bootstrapCmd.Flags().BoolVar(&bootstrap, "bootstrap", false, "Bootstrap network with genesis block")

	// Add subcommands
	rootCmd.AddCommand(bootstrapCmd)
}

func runOrderer(cmd *cobra.Command, args []string) {
	logger.Info("Starting orderer process...")

	// Validate input parameters
	if err := validateOrdererParams(); err != nil {
		logger.Fatalf("Invalid parameters: %v", err)
	}

	// Create orderer instance with MSP files
	o, err := orderer.NewOrderer(mspID, mspPath)
	if err != nil {
		logger.Fatalf("Failed to create orderer: %v", err)
	}

	// Create gRPC server
	server := orderer.NewOrdererServer(o)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal, stopping orderer...")
		cancel()
	}()

	// Start server with context for graceful shutdown
	logger.Infof("Starting orderer server on %s with MSP ID: %s", address, mspID)
	if err := server.StartWithContext(ctx, address); err != nil {
		logger.Fatalf("Failed to start orderer server: %v", err)
	}

	logger.Info("Orderer server stopped gracefully")
}

func runBootstrap(cmd *cobra.Command, args []string) {
	logger.Info("Starting network bootstrap process...")

	// Orderer 인스턴스 생성 (MSP 파일 사용)
	o, err := orderer.NewOrderer(mspID, mspPath)
	if err != nil {
		logger.Fatalf("Failed to create orderer: %v", err)
	}

	// configtx.yaml에서 제네시스 설정 생성 (profile 인자 추가)
	genesisConfig, err := orderer.CreateGenesisConfigFromConfigTx(configTxPath, profile)
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute orderer command: %v", err)
	}
}
