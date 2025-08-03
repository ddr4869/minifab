package channel

import (
	"crypto/x509"
	"encoding/json"
	"sync"
	"time"

	"github.com/ddr4869/minifab/common/blockutil"
	"github.com/ddr4869/minifab/common/cert"
	"github.com/ddr4869/minifab/common/configtx"
	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/config"
	pb_common "github.com/ddr4869/minifab/proto/common"
	pb_orderer "github.com/ddr4869/minifab/proto/orderer"
	"github.com/pkg/errors"
)

// ChainSupport는 Orderer 노드에서 메모리 상 존재하며 채널 정보를 관리한다.
// 지속성 있는 데이터의 경우 Orderer의 파일 시스템에 저장되어야 한다.
type ChainSupport struct {
	SystemChannelInfo *configtx.SystemChannelInfo
	AppChannelConfigs map[string]*configtx.ChannelConfig

	OrdererConfig *config.OrdererCfg
	Mutex         sync.RWMutex
	pb_orderer.UnimplementedOrdererServiceServer
}

func (cs *ChainSupport) GetSystemChannelConfig() *configtx.SystemChannelInfo {
	return cs.SystemChannelInfo
}

func (cs *ChainSupport) LoadSystemChannelConfig(genesisPath string) {
	scc, err := blockutil.LoadSystemChannelConfig(genesisPath)
	if err != nil {
		logger.Panicf("Failed to load system channel config: %v", err)
		return
	}
	cs.SystemChannelInfo = scc
}

func (cs *ChainSupport) LoadExistingChannels(filesystemPath string) {
	logger.Info("🔄 Loading existing channel configurations...")

	appChannelConfigs, err := blockutil.LoadAppChannelConfigs(filesystemPath)
	if err != nil {
		logger.Errorf("Failed to load existing channel configs: %v", err)
		return
	}

	for channelName, config := range appChannelConfigs {
		cs.AppChannelConfigs[channelName] = &configtx.ChannelConfig{
			CC:  config.CC,
			SCC: config.SCC,
		}
	}
}

// check func (h *Handler) ProcessStream(stream ccintf.ChaincodeStream) error
func (cs *ChainSupport) CreateChannel(stream pb_orderer.OrdererService_CreateChannelServer) error {
	cs.Mutex.Lock()
	defer cs.Mutex.Unlock()
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		if err := cs.VerifyChannelCreationEnvelope(msg); err != nil {
			return err
		}
		payload, err := blockutil.UnmarshalPayloadFromProto(msg.Payload)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal payload")
		}

		block, err := blockutil.UnmarshalBlockFromProto(payload.Data)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal block")
		}
		appConfig, err := blockutil.ExtractAppChannelConfigFromBlock(block)
		if err != nil {
			return errors.Wrap(err, "failed to extract app channel config from block")
		}
		logger.Info("appConfig -> ", appConfig)
		logger.Infof("[Orderer] Received app config: %+v", appConfig)
		time.Sleep(3 * time.Second)

		// #TODO : phase 2 - Save config block to the ChainSupport
		appChannelConfig := &configtx.ChannelConfig{
			CC:  appConfig,
			SCC: cs.SystemChannelInfo,
		}
		cs.AppChannelConfigs[payload.Header.ChannelId] = appChannelConfig

		configDataBytes, err := json.Marshal(appChannelConfig)
		if err != nil {
			return errors.Wrap(err, "failed to marshal channel config data")
		}
		appBlock, err := blockutil.GenerateConfigBlock(configDataBytes, payload.Header.ChannelId, cs.OrdererConfig.MSP.GetSigningIdentity())
		if err != nil {
			return errors.Wrap(err, "failed to generate config block")
		}
		if err := blockutil.SaveBlockFile(appBlock, payload.Header.ChannelId, cs.OrdererConfig.FilesystemPath); err != nil {
			return errors.Wrap(err, "failed to save config block")
		}
		// TODO : Envelope 생성 후 전송
		stream.Send(appBlock)
		time.Sleep(3 * time.Second)
		logger.Infof("[Orderer] Sent app block to the peer")

	}
}

func (cs *ChainSupport) GetChannelInfo(channelName string) (*configtx.ChannelConfig, bool) {
	config, exists := cs.AppChannelConfigs[channelName]
	return config, exists
}

func (cs *ChainSupport) ListChannels() []string {
	channels := make([]string, 0, len(cs.AppChannelConfigs))
	for channelName := range cs.AppChannelConfigs {
		channels = append(channels, channelName)
	}
	return channels
}

func (cs *ChainSupport) VerifyChannelCreationEnvelope(envelope *pb_common.Envelope) error {
	Payload, err := blockutil.UnmarshalPayloadFromProto(envelope.Payload)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal payload")
	}
	if Payload.Header.Type != pb_common.MessageType_MESSAGE_TYPE_CONFIG {
		return errors.New("invalid message type")
	}
	if cs.AppChannelConfigs[Payload.Header.ChannelId] != nil {
		return errors.New("channel already exists")
	}

	identity, err := blockutil.GetIdentityFromHeader(Payload.Header)
	if err != nil {
		return errors.Wrap(err, "failed to get identity from header")
	}
	creatorCert, err := x509.ParseCertificate(identity.Creator)
	if err != nil {
		logger.Error("failed to parse certificate from identity")
		return errors.Wrap(err, "failed to parse certificate from identity")
	}

	// #1 : verify sender(client) signature
	ok, err := cert.VerifySignature(creatorCert.PublicKey, envelope.Payload, envelope.Signature)
	if err != nil {
		return errors.Wrap(err, "failed to verify signature")
	}
	if !ok {
		return errors.New("failed to verify signature")
	}
	logger.Infof("[Orderer] Signature verified: %v", ok)

	// #2 : verify certificate chain & MSPID in consortiums
	ok, err = cs.VerifyConsortiumMSP(creatorCert, identity.MspId)
	if err != nil {
		return errors.Wrap(err, "failed to verify rootCACerts")
	}
	if !ok {
		return errors.New("failed to verify rootCACerts")
	}
	return nil
}

func (cs *ChainSupport) VerifyConsortiumMSP(creatorCert *x509.Certificate, mspId string) (bool, error) {
	scc := cs.GetSystemChannelConfig()
	if scc == nil {
		return false, errors.New("system channel config is not loaded")
	}

	for _, consortium := range scc.Consortiums {
		if consortium.ID == mspId {
			logger.Infof("[Orderer] MSPID verified: %s", mspId)
			consortiumCert, err := x509.ParseCertificate(consortium.MSPCaCert)
			if err != nil {
				return false, errors.Wrap(err, "failed to parse certificate")
			}
			if err := creatorCert.CheckSignatureFrom(consortiumCert); err != nil {
				return false, errors.Wrap(err, "failed to verify certificate chain")
			}
			logger.Infof("[Orderer] Certificate chain verified")
			return true, nil
		}
	}
	return false, nil
}
