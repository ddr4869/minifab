package channel

// import (
// 	"github.com/ddr4869/minifab/common/logger"
// 	"github.com/ddr4869/minifab/peer/cli"
// 	"github.com/pkg/errors"
// 	"github.com/spf13/cobra"
// )

// var (
// 	defaultProfile = "testchannel0"
// )

// // var channelCreateCmd = &cobra.Command{
// // 	Use:   "create [channel-name] [profile-name]",
// // 	Short: "새로운 채널을 생성합니다",
// // 	Long:  `지정된 이름으로 새로운 채널을 생성하고 orderer에 알립니다.`,
// // 	Args:  cobra.ExactArgs(2),
// // 	Run: func(cmd *cobra.Command, args []string) {
// // 		channelName := args[0]
// // 		profileName := args[1]
// // 		if profileName == "" {
// // 			profileName = defaultProfile
// // 		}
// // 		if err := cliHandlers.HandleChannelCreateWithProfile(channelName, profileName); err != nil {
// // 			log.Fatalf("Failed to create channel: %v", err)
// // 		}
// // 	},
// // }

// func channelCreateCmd(handler *cli.Handlers) *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:   "create [channel-name] [profile-name]",
// 		Short: "새로운 채널을 생성합니다",
// 		Long:  `지정된 이름으로 새로운 채널을 생성하고 orderer에 알립니다.`,
// 		Args:  cobra.ExactArgs(2),
// 	}

// 	cmd.RunE = func(cmd *cobra.Command, args []string) error {
// 		channelName := args[0]
// 		profileName := args[1]
// 		if profileName == "" {
// 			profileName = defaultProfile
// 		}
// 		return CreateChannelWithProfile(handler, channelName, profileName)
// 	}

// 	return cmd
// }

// // CreateChannelWithProfile creates a channel with specific profile via orderer and then creates it locally
// func CreateChannelWithProfile(handler *cli.Handlers, channelName, profileName string) error {
// 	logger.Infof("[Peer] Creating channel: %s with profile: %s", channelName, profileName)

// 	// 1. First, request channel creation from orderer with profile
// 	if handler.OrdererClient == nil {
// 		return errors.New("orderer client is required for channel creation")
// 	}

// 	if err := handler.OrdererClient.CreateChannelWithProfile(channelName, profileName, "config/configtx.yaml"); err != nil {
// 		return errors.Wrapf(err, "failed to create channel %s via orderer with profile %s", channelName, profileName)
// 	}

// 	// 2. Then create the channel locally
// 	if p.channelManager == nil {
// 		return errors.New("channel manager not initialized")
// 	}

// 	if err := p.channelManager.CreateChannel(channelName, "SampleConsortium", "localhost:7050"); err != nil {
// 		return errors.Wrapf(err, "failed to create local channel %s", channelName)
// 	}

// 	logger.Infof("[Peer] Channel %s created successfully with profile %s", channelName, profileName)
// 	return nil
// }
