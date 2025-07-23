package channel

import (
	"log"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/peer/core"
	"github.com/spf13/cobra"
)

// getChannelJoinCmd는 기존 채널에 참여합니다
func getChannelJoinCmd(peer *core.Peer) *cobra.Command {

	var channelName string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "기존 채널에 참여합니다",
		Long:  `지정된 이름으로 기존 채널에 참여합니다.`,
		Run: func(cmd *cobra.Command, args []string) {
			if channelName == "" {
				log.Fatalf("Channel name is required. Use -c or --channelID flag")
			}

			if err := JoinChannel(peer, channelName); err != nil {
				log.Fatalf("Failed to create channel: %v", err)
			}
		},
	}

	// Fabric CLI 스타일 플래그 추가
	cmd.Flags().StringVarP(&channelName, "channelID", "c", "", "Channel name (required)")
	cmd.MarkFlagRequired("channelID")

	return cmd
}

func JoinChannel(peer *core.Peer, channelName string) error {
	logger.Infof("[Peer] Joining channel: %s", channelName)

	return nil
}
