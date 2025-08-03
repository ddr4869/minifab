package blockutil

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ddr4869/minifab/common/configtx"
	"github.com/ddr4869/minifab/common/logger"
	pb_common "github.com/ddr4869/minifab/proto/common"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

func LoadSystemChannelConfig(blockPath string) (*configtx.SystemChannelInfo, error) {
	Block, err := LoadBlock(blockPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load block")
	}

	if Block.Header.HeaderType != pb_common.BlockType_BLOCK_TYPE_CONFIG {
		return nil, errors.New("block is not a config block")
	}

	return ExtractSystemChannelConfigFromBlock(Block)
}

func LoadAppChannelConfigs(filesystemPath string) (map[string]*configtx.ChannelConfig, error) {
	channelConfigs := make(map[string]*configtx.ChannelConfig)

	if _, err := os.Stat(filesystemPath); os.IsNotExist(err) {
		logger.Infof("Filesystem path does not exist: %s, starting with empty channel configs", filesystemPath)
		return channelConfigs, nil
	}

	entries, err := os.ReadDir(filesystemPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read filesystem directory: %s", filesystemPath)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		channelName := entry.Name()
		blockPath := fmt.Sprintf("%s/%s/blockfile0", filesystemPath, channelName)

		appChannelConfigData, err := LoadChannelConfigDataFromBlock(blockPath)
		if err != nil {
			logger.Warnf("Failed to load channel config data for %s: %v", channelName, err)
			continue
		}

		channelConfigs[channelName] = appChannelConfigData

		logger.Infof("✅ Loaded app channel config for: %s", channelName)
	}

	return channelConfigs, nil
}

// LoadChannelConfigDataFromBlock 블록 파일에서 채널 설정 데이터 로드
func LoadChannelConfigDataFromBlock(blockPath string) (*configtx.ChannelConfig, error) {
	block, err := LoadBlock(blockPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load block")
	}

	if block.Header.HeaderType != pb_common.BlockType_BLOCK_TYPE_CONFIG {
		return nil, errors.New("block is not a config block")
	}
	if len(block.Data.Transactions) == 0 {
		return nil, errors.New("no transactions found in block")
	}

	// Transaction proto로 파싱
	protoTx := &pb_common.Transaction{}
	if err := proto.Unmarshal(block.Data.Transactions[0], protoTx); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal transaction")
	}

	// ChannelConfigData JSON 파싱
	var channelConfigData configtx.ChannelConfig
	if err := json.Unmarshal(protoTx.Payload, &channelConfigData); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal channel config data")
	}
	logger.Info("!! channelConfigData -> ", channelConfigData)
	logger.Infof("✅ Successfully extracted ChannelConfigData from block")
	return &channelConfigData, nil
}

func ExtractSystemChannelConfigFromBlock(block *pb_common.Block) (*configtx.SystemChannelInfo, error) {
	if len(block.Data.Transactions) == 0 {
		return nil, errors.New("no transactions found in block")
	}

	protoTx := &pb_common.Transaction{}
	if err := proto.Unmarshal(block.Data.Transactions[0], protoTx); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal transaction")
	}

	var systemChannelInfo configtx.SystemChannelInfo
	if err := json.Unmarshal(protoTx.Payload, &systemChannelInfo); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal system channel config")
	}

	logger.Infof("✅ Successfully extracted SystemChannelConfig from block")
	return &systemChannelInfo, nil
}

func ExtractAppChannelConfigFromBlock(block *pb_common.Block) (*configtx.AppChannelConfig, error) {
	if len(block.Data.Transactions) == 0 {
		return nil, errors.New("no transactions found in block")
	}

	protoTx := &pb_common.Transaction{}
	if err := proto.Unmarshal(block.Data.Transactions[0], protoTx); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal transaction")
	}

	var systemChannelInfo configtx.AppChannelConfig
	if err := json.Unmarshal(protoTx.Payload, &systemChannelInfo); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal system channel config")
	}

	logger.Infof("✅ Successfully extracted SystemChannelConfig from block")
	return &systemChannelInfo, nil
}

func GetConfigTxFromBlock(block *pb_common.Block) (*pb_common.Transaction, error) {
	if len(block.Data.Transactions) == 0 {
		return nil, errors.New("no transactions found in block")
	}

	tx, err := UnmarshalTransactionFromProto(block.Data.Transactions[0])
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal transaction from block")
	}

	return tx, nil
}
