package channel

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"log"
	"time"

	"github.com/ddr4869/minifab/common/blockutil"
	"github.com/ddr4869/minifab/common/configtx"
	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/peer/core"
	pb_common "github.com/ddr4869/minifab/proto/common"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const cfgFile = "config/configtx.yaml"

// getChannelCreateCmd는 새로운 채널을 생성합니다
func ChannelCreateCmd(peer *core.Peer) *cobra.Command {
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

func CreateChannel(peer *core.Peer, channelName, profileName string) error {

	// #TODO :phase 0 - check peer's identity
	if peer.OrdererClient == nil {
		return errors.New("orderer client is required for channel creation")
	}

	// #phase 1 - create config block
	appConfigBlock, err := generateAppConfigBlock(peer, channelName, profileName)
	if err != nil {
		return errors.Wrap(err, "failed to generate app config block")
	}
	protoData, err := proto.Marshal(appConfigBlock)
	if err != nil {
		return errors.Wrap(err, "failed to marshal genesis block")
	}

	if err := blockutil.SaveBlock(protoData, channelName); err != nil {
		return errors.Wrap(err, "failed to save config block")
	}

	// #phase 2 - send config block to orderer
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	payload := &pb_common.Payload{
		Header: &pb_common.Header{
			Type:      pb_common.MessageType_MESSAGE_TYPE_CONFIG,
			ChannelId: channelName,
			Timestamp: timestamppb.Now(),
		},
		Data: protoData,
	}

	stream, err := peer.OrdererClient.GetClient().CreateChannel(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to create channel")
	}

	sig, err := peer.PeerConfig.Msp.GetSigningIdentity().Sign(rand.Reader, payload.Data, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to sign payload")
	}

	stream.Send(&pb_common.Envelope{
		Payload:   payload,
		Signature: sig,
	})
	response, err := stream.Recv()
	if err != nil {
		return errors.Wrapf(err, "failed to receive response")
	}
	logger.Info("✅ Broadcast response: ", response)

	return nil
}

func generateAppConfigBlock(peer *core.Peer, channelName, profileName string) (*pb_common.ConfigBlock, error) {
	appConfig, err := CreateAppConfigFromConfigTx(cfgFile, profileName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create app config")
	}

	appConfigBytes, err := json.Marshal(appConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal app config")
	}

	return blockutil.GenerateConfigBlock(appConfigBytes, channelName, peer.PeerConfig.Msp.GetSigningIdentity())
}

func CreateAppConfigFromConfigTx(configTxPath string, profile string) (*configtx.ChannelConfig, error) {

	configTx, err := configtx.ConvertConfigtx(configTxPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert configtx")
	}
	// ConfigTxYAML을 ConfigTx로 변환
	genesisConfig, err := configTx.GetAppChannelProfile(profile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert configtx to genesis config")
	}

	logger.Infof("Successfully loaded configuration from %s with profile %s", configTxPath, profile)

	return &genesisConfig.Application, nil
}
