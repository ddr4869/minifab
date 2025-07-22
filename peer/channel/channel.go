package channel

import (
	"log"

	"github.com/spf13/cobra"
)

var (
	OrdererAddress string
	PeerID         string
	ChaincodePath  string
	MspID          string
	MspPath        string
)

const defaultProfile = "testchannel0"

// GetChannelCommand returns the channel command with all subcommands
func Cmd() *cobra.Command {
	channelCmd := &cobra.Command{
		Use:   "channel",
		Short: "채널 관련 작업을 수행합니다",
		Long:  `채널 생성, 참여, 목록 조회 등의 채널 관련 작업을 수행합니다.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			common.initializePeer(cmd, args)
		},
	}

	// 서브커맨드들 추가
	channelCmd.AddCommand(getChannelCreateCmd())
	channelCmd.AddCommand(getChannelJoinCmd())
	channelCmd.AddCommand(getChannelListCmd())

	return channelCmd
}

// getChannelCreateCmd는 새로운 채널을 생성합니다
func getChannelCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create [channel-name] [profile-name]",
		Short: "새로운 채널을 생성합니다",
		Long:  `지정된 이름으로 새로운 채널을 생성하고 orderer에 알립니다.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			channelName := args[0]
			profileName := args[1]
			if profileName == "" {
				profileName = defaultProfile
			}
			if err := CreateChannelWithProfile(channelName, profileName); err != nil {
				log.Fatalf("Failed to create channel: %v", err)
			}
		},
	}
}

// getChannelJoinCmd는 기존 채널에 참여합니다
func getChannelJoinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "join [channel-name]",
		Short: "기존 채널에 참여합니다",
		Long:  `지정된 이름의 기존 채널에 이 peer를 참여시킵니다.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			channelName := args[0]
			if err := JoinChannel(channelName); err != nil {
				log.Fatalf("Failed to join channel: %v", err)
			}
		},
	}
}

// getChannelListCmd는 사용 가능한 채널 목록을 보여줍니다
func getChannelListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "사용 가능한 채널 목록을 조회합니다",
		Long:  `현재 peer가 알고 있는 모든 채널의 목록을 표시합니다.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := ListChannels(); err != nil {
				log.Fatalf("Failed to list channels: %v", err)
			}
		},
	}
}
