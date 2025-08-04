package channel

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"sync"

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
		if err == io.EOF {
			logger.Infof("[Orderer] Client disconnected")
			return nil
		}
		if err != nil {
			logger.Errorf("[Orderer] Failed to receive message: %v", err)
			return err
		}
		if err := cs.VerifyChannelCreationEnvelope(msg); err != nil {
			cs.sendErrorResponse(stream, pb_common.Status_INVALID_SIGNATURE, fmt.Sprintf("Envelope verification failed: %v", err))
			return err
		}

		payload, err := blockutil.UnmarshalPayloadFromProto(msg.Payload)
		if err != nil {
			cs.sendErrorResponse(stream, pb_common.Status_INVALID_TRANSACTION_FORMAT, fmt.Sprintf("Failed to unmarshal payload: %v", err))
			return err
		}

		block, err := blockutil.UnmarshalBlockFromProto(payload.Data)
		if err != nil {
			cs.sendErrorResponse(stream, pb_common.Status_INVALID_BLOCK, fmt.Sprintf("Failed to unmarshal block: %v", err))
			return err
		}

		appConfig, err := blockutil.ExtractAppChannelConfigFromBlock(block)
		if err != nil {
			cs.sendErrorResponse(stream, pb_common.Status_INVALID_BLOCK, fmt.Sprintf("Failed to extract app channel config: %v", err))
			return err
		}
		logger.Infof("[Orderer] Received app config: %+v", appConfig)

		if _, exists := cs.AppChannelConfigs[payload.Header.ChannelId]; exists {
			cs.sendErrorResponse(stream, pb_common.Status_ALREADY_EXISTS, fmt.Sprintf("Channel already exists: %s", payload.Header.ChannelId))
			return errors.New("channel already exists")
		}

		appChannelConfig := &configtx.ChannelConfig{
			CC:  appConfig,
			SCC: cs.SystemChannelInfo,
		}
		cs.AppChannelConfigs[payload.Header.ChannelId] = appChannelConfig

		configDataBytes, err := json.Marshal(appChannelConfig)
		if err != nil {
			cs.sendErrorResponse(stream, pb_common.Status_INTERNAL_ERROR, fmt.Sprintf("Failed to marshal channel config data: %v", err))
			return err
		}

		appBlock, err := blockutil.GenerateConfigBlock(configDataBytes, payload.Header.ChannelId, cs.OrdererConfig.MSP.GetSigningIdentity())
		if err != nil {
			cs.sendErrorResponse(stream, pb_common.Status_INTERNAL_ERROR, fmt.Sprintf("Failed to generate config block: %v", err))
			return err
		}

		if err := blockutil.SaveBlockFile(appBlock, payload.Header.ChannelId, cs.OrdererConfig.FilesystemPath); err != nil {
			cs.sendErrorResponse(stream, pb_common.Status_LEDGER_ERROR, fmt.Sprintf("Failed to save config block: %v", err))
			return err
		}
		if err := cs.sendSuccessResponse(stream, appBlock, payload.Header.ChannelId); err != nil {
			return err
		}
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

func (cs *ChainSupport) sendErrorResponse(stream pb_orderer.OrdererService_CreateChannelServer, status pb_common.Status, errMsg string) {
	logger.Errorf("[Orderer] %s", errMsg)
	response := &pb_orderer.BroadcastResponse{
		Status: status,
		Block:  nil,
	}
	if sendErr := stream.Send(response); sendErr != nil {
		logger.Errorf("[Orderer] Failed to send error response: %v", sendErr)
	}
}

func (cs *ChainSupport) sendSuccessResponse(stream pb_orderer.OrdererService_CreateChannelServer, block *pb_common.Block, channelId string) error {
	response := &pb_orderer.BroadcastResponse{
		Status: pb_common.Status_OK,
		Block:  block,
	}
	if err := stream.Send(response); err != nil {
		logger.Errorf("[Orderer] Failed to send success response: %v", err)
		return err
	}
	logger.Infof("[Orderer] Successfully created channel: %s", channelId)
	return nil
}
