package blockutil

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"os"
	"time"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
	pb_common "github.com/ddr4869/minifab/proto/common"
	"github.com/pkg/errors"
)

func GenerateConfigBlock(config []byte, channelName string, signer msp.SigningIdentity) (*pb_common.ConfigBlock, error) {
	header := &pb_common.BlockHeader{
		Number:       0,
		PreviousHash: nil,
		HeaderType:   pb_common.BlockType_BLOCK_TYPE_CONFIG,
	}

	blockData := &pb_common.BlockData{
		Transactions: [][]byte{
			config,
		},
	}
	signature, err := signer.Sign(rand.Reader, config, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign config")
	}

	metadata := &pb_common.BlockMetadata{
		Signature:        signature,
		ValidationBitmap: []byte{1},
		AccumulatedHash:  []byte{},
	}

	block := &pb_common.Block{
		Header:   header,
		Data:     blockData,
		Metadata: metadata,
	}

	header.CurrentBlockHash = CalculateBlockHash(block)

	genesisBlock := &pb_common.ConfigBlock{
		Block:       block,
		ChannelId:   channelName,
		StoredAt:    time.Now().Format(time.RFC3339),
		IsCommitted: true,
		BlockHash:   fmt.Sprintf("%x", header.CurrentBlockHash),
	}

	return genesisBlock, nil
}

// SaveBlock은 블록 데이터를 파일로 저장하는 함수
// 만약 폴더가 존재하지 않으면 폴더를 생성
// 현재 해당 채널에 쌓여있는 블록 파일 개수를 카운트하여 그 개수를 블록 번호로 사용
func SaveBlock(blockData []byte, channelName string) error {
	// 먼저 폴더 생성
	channelDir := fmt.Sprintf("./blocks/%s", channelName)
	if err := os.MkdirAll(channelDir, 0755); err != nil {
		return errors.Wrapf(err, "failed to create directory: %s", channelDir)
	}

	blockNumber := 0
	for {
		blockFile := fmt.Sprintf("%s/blockfile%d", channelDir, blockNumber)
		if _, err := os.Stat(blockFile); os.IsNotExist(err) {
			break
		}
		blockNumber++
	}

	// 블록 파일 저장
	blockFilePath := fmt.Sprintf("%s/blockfile%d", channelDir, blockNumber)
	if err := os.WriteFile(blockFilePath, blockData, 0644); err != nil {
		return errors.Wrapf(err, "failed to write block file: %s", blockFilePath)
	}

	logger.Infof("✅ Block %d saved successfully at %s", blockNumber, blockFilePath)
	return nil
}

func CalculateBlockHash(block *pb_common.Block) []byte {
	if block == nil {
		return nil
	}

	// TODO: 블록 해시 계산 로직 추가
	hash := sha256.New()
	return hash.Sum(block.Header.PreviousHash)
}
