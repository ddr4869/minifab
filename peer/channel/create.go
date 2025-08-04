package channel

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"log"

	"github.com/ddr4869/minifab/common/blockutil"
	"github.com/ddr4869/minifab/common/configtx"
	"github.com/ddr4869/minifab/common/msp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/peer/core"
	pb_common "github.com/ddr4869/minifab/proto/common"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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
	appCfgBytes, err := blockutil.MarshalBlockToProto(appConfigBlock)
	if err != nil {
		return errors.Wrap(err, "failed to marshal genesis block")
	}

	// #phase 2 - send config block to orderer
	envelope, err := ProcessConfigBlock(peer.Peer.MSP.GetSigningIdentity(), channelName, appCfgBytes)
	if err != nil {
		return errors.Wrap(err, "failed to create payload")
	}
	block, err := peer.OrdererClient.Send(envelope)
	if err != nil {
		return errors.Wrapf(err, "failed to send envelope")
	}

	// #phase 3 - save config block
	// TODO : Committer 작업 적용 후 저장
	if err := blockutil.SaveBlockFile(block.Block, channelName, peer.Peer.FilesystemPath); err != nil {
		return errors.Wrap(err, "failed to save config block")
	}
	logger.Info("✅ Broadcast Success")

	return nil
}

func ProcessConfigBlock(signer msp.SigningIdentity, channelName string, data []byte) (*pb_common.Envelope, error) {

	header, err := blockutil.CreateHeader(signer, pb_common.MessageType_MESSAGE_TYPE_CONFIG, channelName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create header")
	}
	header.Timestamp = timestamppb.Now()

	payload, err := blockutil.CreatePayload(header, data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create payload")
	}
	payloadBytes, err := blockutil.MarshalPayloadToProto(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal payload")
	}

	payloadHash := sha256.Sum256(payloadBytes)
	sig, err := signer.Sign(rand.Reader, payloadHash[:], nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to sign payload")
	}

	envelope, err := blockutil.CreateEnvelope(payload, sig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create envelope")
	}
	return envelope, nil
}

func generateAppConfigBlock(peer *core.Peer, channelName, profileName string) (*pb_common.Block, error) {
	appConfig, err := CreateAppConfigFromConfigTx(cfgFile, profileName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create app config")
	}

	appConfigBytes, err := json.Marshal(appConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal app config")
	}

	return blockutil.GenerateConfigBlock(appConfigBytes, channelName, peer.Peer.MSP.GetSigningIdentity())
}

func CreateAppConfigFromConfigTx(configTxPath string, profile string) (*configtx.AppChannelConfig, error) {
	// yaml.Unmarshal로 confitx.yaml로 불러온 후 proto.Marshal로 직렬화 해야함
	// 직렬화된 데이터를 사용하여 채널 구성 생성
	// 채널 구성 생성 후 채널 구성 반환
	// (core - func NewChannelGroup(conf *genesisconfig.Profile) 참고)
	// 그게 아니면 Orderer 쪽에서 그냥
	ccfg, err := configtx.ConvertConfigtx(configTxPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert configtx")
	}

	genesisConfig, err := ccfg.GetAppChannelProfile(profile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert configtx to genesis config")
	}

	logger.Infof("Successfully loaded configuration from %s with profile %s", configTxPath, profile)

	return &genesisConfig.Application, nil
}
