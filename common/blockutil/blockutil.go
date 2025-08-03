package blockutil

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/ddr4869/minifab/common/configtx"
	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/common/msp"
	pb_common "github.com/ddr4869/minifab/proto/common"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

// 추후 다른 TYPE의 트랜잭션까지 Handling
func GenerateConfigBlock(channelConfig []byte, channelName string, signer msp.SigningIdentity) (*pb_common.Block, error) {

	header := &pb_common.BlockHeader{
		Number:       0,
		PreviousHash: nil,
		HeaderType:   pb_common.BlockType_BLOCK_TYPE_CONFIG,
	}

	// #1 generate tx, signature - peer identity
	id := &pb_common.Identity{
		Creator: signer.GetCertificate().Raw,
		MspId:   signer.GetIdentifier().Mspid,
	}
	protoId, err := proto.Marshal(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal identity")
	}
	idhash := sha256.Sum256(protoId)
	signature, err := signer.Sign(rand.Reader, idhash[:], nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign config")
	}

	tx := &pb_common.Transaction{
		Payload:   channelConfig,
		Identity:  id,
		Timestamp: time.Now().Unix(),
	}
	txID, err := CalculateTxHash(tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to calculate transaction hash")
	}
	tx.TxId = txID
	tx.Signature = signature

	protoTx, err := proto.Marshal(tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal transaction")
	}
	blockData := &pb_common.BlockData{
		Transactions: [][]byte{
			protoTx,
		},
	}

	// # 2 blcok metadata
	metadata := &pb_common.BlockMetadata{
		// Signature: signature,
		Identity: &pb_common.Identity{
			Creator: signer.GetCertificate().Raw,
			MspId:   signer.GetIdentifier().Mspid,
		},
		ValidationBitmap: []byte{1}, // TODO
		AccumulatedHash:  []byte{},  // TODO
	}

	block := &pb_common.Block{
		Header:   header,
		Data:     blockData,
		Metadata: metadata,
	}
	header.CurrentBlockHash = CalculateBlockHash(block)

	return block, nil
}

// SaveBlock은 블록 데이터를 파일로 저장하는 함수
// 만약 폴더가 존재하지 않으면 폴더를 생성
// 현재 해당 채널에 쌓여있는 블록 파일 개수를 카운트하여 그 개수를 블록 번호로 사용
func SaveBlockFile(blockProto *pb_common.Block, channelName string, FilesystemPath string) error {
	// 먼저 폴더 생성
	channelDir := fmt.Sprintf("%s/%s", FilesystemPath, channelName)
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

	blockData, err := MarshalBlockToProto(blockProto)
	if err != nil {
		return errors.Wrap(err, "failed to marshal block to proto")
	}
	blockFilePath := fmt.Sprintf("%s/blockfile%d", channelDir, blockNumber)
	if err := os.WriteFile(blockFilePath, blockData, 0644); err != nil {
		return errors.Wrapf(err, "failed to write block file: %s", blockFilePath)
	}
	logger.Infof("✅ Block %d saved successfully at %s", blockNumber, blockFilePath)
	return nil
}

func LoadBlock(blockPath string) (*pb_common.Block, error) {
	blockData, err := os.ReadFile(blockPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read block file: %s", blockPath)
	}

	return UnmarshalBlockFromProto(blockData)
}

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

func CalculateTxHash(tx *pb_common.Transaction) (string, error) {
	txBytes, err := proto.Marshal(tx)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal transaction for hash")
	}
	hash := sha256.Sum256(txBytes)
	return hex.EncodeToString(hash[:]), nil
}

// TODO : 블록 해시 계산 로직 추가
func CalculateBlockHash(block *pb_common.Block) []byte {
	if block == nil {
		return nil
	}
	hash := sha256.New()
	return hash.Sum(block.Header.PreviousHash)
}
