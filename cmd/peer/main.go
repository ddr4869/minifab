package main

import (
	"log"
	"os"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/peer/channel"
	"github.com/ddr4869/minifab/peer/core"
	"github.com/spf13/cobra"
)

// rootCmd는 peer의 루트 명령어를 나타냅니다
var rootCmd = &cobra.Command{
	Use:   "peer",
	Short: "Custom Fabric Peer Node",
	Long: `Custom Fabric Peer Node - 블록체인 네트워크에서 트랜잭션을 처리하고 
체인코드를 실행하는 peer 노드입니다.`,
}

// transactionCmd는 트랜잭션을 제출합니다
var transactionCmd = &cobra.Command{
	Use:   "transaction [channel-name] [payload]",
	Short: "지정된 채널에 트랜잭션을 제출합니다",
	Long:  `지정된 채널에 새로운 트랜잭션을 생성하고 orderer에 제출합니다.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		channelName := args[0]
		payload := []byte(args[1])
		if _, err := channel.SubmitTransaction(channelName, payload); err != nil {
			log.Fatalf("Failed to submit transaction: %v", err)
		}
	},
}

func init() {
	// Initialize logger with development config for CLI
	if err := logger.InitializeDevelopment(); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	// 전역 플래그 정의
	rootCmd.PersistentFlags().StringVar(&core.OrdererAddress, "orderer", "localhost:7050", "Orderer server address")
	rootCmd.PersistentFlags().StringVar(&core.PeerID, "id", "peer0", "Peer ID")
	rootCmd.PersistentFlags().StringVar(&core.ChaincodePath, "chaincode", "./chaincode", "Chaincode path")
	rootCmd.PersistentFlags().StringVar(&core.MspID, "mspid", "Org1MSP", "MSP ID for peer")
	rootCmd.PersistentFlags().StringVar(&core.MspPath, "mspdir", "/Users/mac/go/src/github.com/custom-fabric/ca/ca-client/peer0/msp", "Path to MSP directory with certificates")

	// 서브커맨드 추가
	rootCmd.AddCommand(channel.Cmd())
	rootCmd.AddCommand(transactionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Errorf("Command execution failed: %v", err)
		os.Exit(1)
	}
}
