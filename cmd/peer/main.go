package main

import (
	"os"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/peer/channel"
	"github.com/spf13/cobra"
)

// rootCmd는 peer의 루트 명령어를 나타냅니다
var rootCmd = &cobra.Command{
	Use:   "peer",
	Short: "Custom Fabric Peer Node",
	Long: `Custom Fabric Peer Node - 블록체인 네트워크에서 트랜잭션을 처리하고 
체인코드를 실행하는 peer 노드입니다.`,
}

func init() {
	// Initialize logger with development config for CLI
	if err := logger.InitializeDevelopment(); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	rootCmd.AddCommand(channel.Cmd())

}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Errorf("Command execution failed: %v", err)
		os.Exit(1)
	}
}
