package channel

import (
	"context"
	"log"
	"time"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/peer/core"
	"github.com/ddr4869/minifab/proto"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// getChannelCreateCmd는 새로운 채널을 생성합니다
func getChannelCreateCmd(peer *core.Peer) *cobra.Command {
	var channelName string
	var profileName string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "새로운 채널을 생성합니다",
		Long:  `지정된 이름으로 새로운 채널을 생성하고 orderer에 알립니다.`,
		Run: func(cmd *cobra.Command, args []string) {
			if channelName == "" {
				log.Fatalf("Channel name is required. Use -c or --channelID flag")
			}

			if profileName == "" {
				profileName = "testchannel0"
			}

			if err := CreateChannel(peer, channelName, profileName); err != nil {
				log.Fatalf("Failed to create channel: %v", err)
			}
		},
	}

	// Fabric CLI 스타일 플래그 추가
	cmd.Flags().StringVarP(&channelName, "channelID", "c", "", "Channel name (required)")
	cmd.Flags().StringVarP(&profileName, "profile", "p", "testchannel0", "Profile name for channel creation")
	cmd.MarkFlagRequired("channelID")

	return cmd
}

// CreateChannel creates a channel with specific profile via orderer and then creates it locally
func CreateChannel(peer *core.Peer, channelName, profileName string) error {
	logger.Infof("[Peer] Creating channel: %s with profile: %s", channelName, profileName)

	if peer.OrdererClient == nil {
		return errors.New("orderer client is required for channel creation")
	}

	// 직접 gRPC 호출로 채널 생성
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &proto.ChannelRequest{
		ChannelName:  channelName,
		ProfileName:  profileName,
		ConfigtxPath: "config/configtx.yaml",
	}

	// OrdererClient에서 직접 proto 클라이언트에 접근
	client := peer.OrdererClient.GetClient()
	resp, err := client.CreateChannel(ctx, req)
	if err != nil {
		return errors.Wrapf(err, "failed to create channel %s via orderer with profile %s", channelName, profileName)
	}

	if resp.Status != proto.StatusCode_OK {
		return errors.Errorf("channel creation failed: %s", resp.Message)
	}

	logger.Infof("[Peer] Channel %s created successfully with profile %s", channelName, profileName)
	return nil
}
