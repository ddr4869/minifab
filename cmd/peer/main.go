package main

import (
	"log"
	"os"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/peer"
	"github.com/spf13/cobra"
)

var (
	ordererAddress string
	peerID         string
	chaincodePath  string
	mspID          string
	mspPath        string

	// 전역적으로 사용될 CLI 핸들러
	cliHandlers *peer.CLIHandlers
)

const defaultProfile = "testchannel0"

// rootCmd는 peer의 루트 명령어를 나타냅니다
var rootCmd = &cobra.Command{
	Use:   "peer",
	Short: "Custom Fabric Peer Node",
	Long: `Custom Fabric Peer Node - 블록체인 네트워크에서 트랜잭션을 처리하고 
체인코드를 실행하는 peer 노드입니다.`,
	PersistentPreRun: initializePeer,
}

// channelCmd는 채널 관련 명령어들을 처리합니다
var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "채널 관련 작업을 수행합니다",
	Long:  `채널 생성, 참여, 목록 조회 등의 채널 관련 작업을 수행합니다.`,
}

// channelCreateCmd는 새로운 채널을 생성합니다
var channelCreateCmd = &cobra.Command{
	Use:   "create [channel-name]",
	Short: "새로운 채널을 생성합니다",
	Long:  `지정된 이름으로 새로운 채널을 생성하고 orderer에 알립니다.`,
	Args:  cobra.ExactArgs(1),
	Run:   runChannelCreate,
}

// channelJoinCmd는 기존 채널에 참여합니다
var channelJoinCmd = &cobra.Command{
	Use:   "join [channel-name]",
	Short: "기존 채널에 참여합니다",
	Long:  `지정된 이름의 기존 채널에 이 peer를 참여시킵니다.`,
	Args:  cobra.ExactArgs(1),
	Run:   runChannelJoin,
}

// channelListCmd는 사용 가능한 채널 목록을 보여줍니다
var channelListCmd = &cobra.Command{
	Use:   "list",
	Short: "사용 가능한 채널 목록을 조회합니다",
	Long:  `현재 peer가 알고 있는 모든 채널의 목록을 표시합니다.`,
	Run:   runChannelList,
}

// transactionCmd는 트랜잭션을 제출합니다
var transactionCmd = &cobra.Command{
	Use:   "transaction [channel-name] [payload]",
	Short: "지정된 채널에 트랜잭션을 제출합니다",
	Long:  `지정된 채널에 새로운 트랜잭션을 생성하고 orderer에 제출합니다.`,
	Args:  cobra.ExactArgs(2),
	Run:   runTransaction,
}

func init() {
	// Initialize logger with development config for CLI
	if err := logger.InitializeDevelopment(); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	// 전역 플래그 정의
	rootCmd.PersistentFlags().StringVar(&ordererAddress, "orderer", "localhost:7050", "Orderer server address")
	rootCmd.PersistentFlags().StringVar(&peerID, "id", "peer0", "Peer ID")
	rootCmd.PersistentFlags().StringVar(&chaincodePath, "chaincode", "./chaincode", "Chaincode path")
	rootCmd.PersistentFlags().StringVar(&mspID, "mspid", "Org1MSP", "MSP ID for peer")
	rootCmd.PersistentFlags().StringVar(&mspPath, "mspdir", "/Users/mac/go/src/github.com/custom-fabric/ca/ca-client/peer0/msp", "Path to MSP directory with certificates")

	// 서브커맨드 추가
	rootCmd.AddCommand(channelCmd)
	rootCmd.AddCommand(transactionCmd)

	// channel 서브커맨드들 추가
	channelCmd.AddCommand(channelCreateCmd)
	channelCmd.AddCommand(channelJoinCmd)
	channelCmd.AddCommand(channelListCmd)
}

// initializePeer는 모든 명령어 실행 전에 peer와 orderer client를 초기화합니다
func initializePeer(cmd *cobra.Command, args []string) {
	// Peer 인스턴스 생성 (fabric-ca 인증서 사용)
	p := peer.NewPeerWithMSPFiles(peerID, chaincodePath, mspID, mspPath)

	// Orderer 클라이언트 생성
	ordererClient, err := peer.NewOrdererClient(ordererAddress)
	if err != nil {
		log.Fatalf("Failed to create orderer client: %v", err)
	}

	// CLI 핸들러 생성
	cliHandlers = peer.NewCLIHandlers(p, ordererClient)
}

func runChannelCreate(cmd *cobra.Command, args []string) {
	channelName := args[0]
	// if err := cliHandlers.HandleChannelCreate(channelName, ordererAddress); err != nil {
	// 	log.Fatalf("Failed to create channel: %v", err)
	// }
	if err := cliHandlers.HandleChannelCreateWithProfile(channelName, defaultProfile); err != nil {
		log.Fatalf("Failed to create channel: %v", err)
	}
}

func runChannelJoin(cmd *cobra.Command, args []string) {
	channelName := args[0]
	if err := cliHandlers.HandleChannelJoin(channelName); err != nil {
		log.Fatalf("Failed to join channel: %v", err)
	}
}

func runChannelList(cmd *cobra.Command, args []string) {
	if err := cliHandlers.HandleChannelList(); err != nil {
		log.Fatalf("Failed to list channels: %v", err)
	}
}

func runTransaction(cmd *cobra.Command, args []string) {
	channelName := args[0]
	payload := []byte(args[1])
	if err := cliHandlers.HandleTransaction(channelName, payload); err != nil {
		log.Fatalf("Failed to submit transaction: %v", err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Errorf("Command execution failed: %v", err)
		os.Exit(1)
	}
}
