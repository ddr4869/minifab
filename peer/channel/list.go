package channel

import (
	"log"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/peer/core"
	"github.com/spf13/cobra"
)

// getChannelListCmd는 채널 목록을 조회합니다
func getChannelListCmd(peer *core.Peer) *cobra.Command {

	var channelName string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "사용 가능한 채널 목록을 조회합니다",
		Long:  `현재 peer가 알고 있는 모든 채널의 목록을 표시합니다.`,
		Run: func(cmd *cobra.Command, args []string) {
			if channelName == "" {
				log.Fatalf("Channel name is required. Use -c or --channelID flag")
			}

			if err := ListChannels(peer, channelName); err != nil {
				log.Fatalf("Failed to create channel: %v", err)
			}
		},
	}

	// Fabric CLI 스타일 플래그 추가
	cmd.Flags().StringVarP(&channelName, "channelID", "c", "", "Channel name (required)")
	cmd.MarkFlagRequired("channelID")

	return cmd
}

func ListChannels(peer *core.Peer, channelName string) error {
	logger.Infof("[Peer] ListChannels channel: %s", channelName)

	return nil
}
