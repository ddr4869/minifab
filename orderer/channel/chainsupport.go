package channel

import (
	"encoding/json"
	"encoding/pem"
	"sync"
	"time"

	"github.com/ddr4869/minifab/common/blockutil"
	"github.com/ddr4869/minifab/common/configtx"
	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/config"
	pb_common "github.com/ddr4869/minifab/proto/common"
	pb_orderer "github.com/ddr4869/minifab/proto/orderer"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

// ChainSupport는 Orderer 노드에서 메모리 상 존재하며 채널 정보를 관리한다.
// 지속성 있는 데이터의 경우 Orderer의 파일 시스템에 저장되어야 한다.
type ChainSupport struct {
	SystemChannelConfig *configtx.SystemChannelConfig
	AppChannelConfigs   map[string]*configtx.ChannelConfig

	OrdererConfig *config.OrdererCfg
	Mutex         sync.RWMutex
	pb_orderer.UnimplementedOrdererServiceServer
}

func (cs *ChainSupport) GetSystemChannelConfig() *configtx.SystemChannelConfig {
	return cs.SystemChannelConfig
}

func (cs *ChainSupport) LoadSystemChannelConfig(genesisPath string) {
	scc, err := blockutil.LoadSystemChannelConfig(genesisPath)
	if err != nil {
		logger.Errorf("Failed to load system channel config: %v", err)
		return
	}
	cs.SystemChannelConfig = scc
}

func (cs *ChainSupport) CreateChannel(stream pb_orderer.OrdererService_CreateChannelServer) error {
	cs.Mutex.Lock()
	defer cs.Mutex.Unlock()
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		// channelName := msg.Payload.Header.ChannelId
		if msg.Payload.Header.Type != pb_common.MessageType_MESSAGE_TYPE_CONFIG {
			return errors.New("invalid message type")
		}
		block, _ := pem.Decode(msg.Signature)
		if block == nil {
			logger.Errorf("failed to decode PEM block from directory %s", msg.Signature)
			return errors.Errorf("failed to decode PEM block from directory %s", msg.Signature)
		}

		// verify signature - consortiums
		// msg.Signature <->

		cfgBlock := &pb_common.ConfigBlock{}
		if err := proto.Unmarshal(msg.Payload.Data, cfgBlock); err != nil {
			return errors.Wrap(err, "failed to unmarshal block")
		}

		appConfig := &configtx.ChannelConfig{}
		for _, tx := range cfgBlock.Block.Data.Transactions {
			logger.Infof("[Orderer] Received config block: %s", tx)
			err = json.Unmarshal(tx, appConfig)
			if err != nil {
				return errors.Wrap(err, "failed to unmarshal app config")
			}
			logger.Infof("[Orderer] Received app config: %s", appConfig)
		}

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
