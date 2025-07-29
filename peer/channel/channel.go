package channel

import (
	"log"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/peer/core"
	"github.com/spf13/cobra"
)

var (
	OrdererAddress string
	PeerID         string
	ChaincodePath  string
	MspID          string
	MspPath        string
)

// GetChannelCommand returns the channel command with all subcommands
func Cmd() *cobra.Command {

	channelCmd := &cobra.Command{
		Use:   "channel",
		Short: "채널 관련 작업을 수행합니다",
		Long:  `채널 생성, 참여, 목록 조회 등의 채널 관련 작업을 수행합니다.`,
	}

	flags := channelCmd.PersistentFlags()
	flags.StringVar(&OrdererAddress, "orderer", "localhost:7050", "Orderer server address")
	flags.StringVar(&PeerID, "id", "peer0", "Peer ID")
	flags.StringVar(&ChaincodePath, "chaincode", "./chaincode", "Chaincode path")
	flags.StringVar(&MspID, "mspid", "Org1MSP", "MSP ID for peer")
	flags.StringVar(&MspPath, "mspdir", "/Users/mac/go/src/github.com/custom-fabric/ca/ca-client/peer0/msp", "Path to MSP directory with certificates")

	peer, err := core.NewPeer(PeerID, MspID, MspPath, OrdererAddress)
	if err != nil {
		log.Fatalf("Failed to create peer: %v", err)
	}

	// peer 로그 출력
	logger.Infof("✅ Successfully created peer: %v", peer)
	logger.Infof("✅ MSP ID: %s", peer.PeerConfig.Msp.GetSigningIdentity().GetIdentifier().Mspid)
	logger.Infof("✅ MSP ID: %s", peer.PeerConfig.Msp.GetSigningIdentity().GetIdentifier().Id)

	channelCmd.AddCommand(getChannelCreateCmd(peer))
	channelCmd.AddCommand(getChannelJoinCmd(peer))
	channelCmd.AddCommand(getChannelListCmd(peer))

	return channelCmd
}
