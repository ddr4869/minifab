package channel

import (
	"crypto/x509"
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

// check func (h *Handler) ProcessStream(stream ccintf.ChaincodeStream) error
func (cs *ChainSupport) CreateChannel(stream pb_orderer.OrdererService_CreateChannelServer) error {
	cs.Mutex.Lock()
	defer cs.Mutex.Unlock()
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		Payload, err := blockutil.UnmarshalPayloadFromProto(msg.Payload)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal payload")
		}
		// channelName := msg.Payload.Header.ChannelId
		if Payload.Header.Type != pb_common.MessageType_MESSAGE_TYPE_CONFIG {
			return errors.New("invalid message type")
		}

		// Identity에서 인증서 파싱
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
		ok, err := cert.VerifySignature(creatorCert.PublicKey, msg.Payload, msg.Signature)
		if err != nil {
			return errors.Wrap(err, "failed to verify signature")
		}
		if !ok {
			return errors.New("failed to verify signature")
		}
		logger.Infof("[Orderer] Signature verified: %v", ok)

		// #2 : verify rootCACerts

		// #3 : verify sender MSPID in consortiums

		// finish verify
		block, err := blockutil.UnmarshalBlockFromProto(Payload.Data)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal block")
		}
		// 새로운 함수 사용하여 AppChannelConfig 추출
		appConfig, err := blockutil.ExtractAppChannelConfigFromBlock(block)
		if err != nil {
			return errors.Wrap(err, "failed to extract app channel config from block")
		}
		logger.Infof("[Orderer] Received app config: %+v", appConfig)

		// #TODO : phase 1 - check if channel already exists
		// #TODO : phase 2 - create config block
		// #TODO : phase 3 - send config block to orderer
		// #TODO : phase 4 - save config block to file
		// #TODO : phase 5 - send config block to orderer
		// #TODO : phase 6 - save config block to file
		time.Sleep(3 * time.Second)

		stream.Send(&pb_common.Block{
			Header: &pb_common.BlockHeader{
				Number: 1,
			},
		})

	}
}
