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

func LoadBlock(blockPath string) (*pb_common.Block, error) {
	blockData, err := os.ReadFile(blockPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read block file: %s", blockPath)
	}
	return UnmarshalBlockFromProto(blockData)
}

// GenerateConfigBlock 설정 블록 생성 (기존 함수들을 활용하여 리팩토링)
func GenerateConfigBlock(channelConfig []byte, channelName string, signer msp.SigningIdentity) (*pb_common.Block, error) {
	identity := &pb_common.Identity{
		Creator: signer.GetCertificate().Raw,
		MspId:   signer.GetIdentifier().Mspid,
	}
	tx := &pb_common.Transaction{
		Payload:   channelConfig,
		Identity:  identity,
		Timestamp: time.Now().Unix(),
	}
	txID, err := CalculateTxHash(tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to calculate transaction hash")
	}
	tx.TxId = txID
	identityBytes, err := MarshalIdentityToProto(identity)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal identity")
	}
	identityHash := sha256.Sum256(identityBytes)
	signature, err := signer.Sign(rand.Reader, identityHash[:], nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign config")
	}
	tx.Signature = signature
	protoTx, err := MarshalTransactionToProto(tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal transaction")
	}
	header := &pb_common.BlockHeader{
		Number:       0,
		PreviousHash: nil,
		HeaderType:   pb_common.BlockType_BLOCK_TYPE_CONFIG,
	}
	blockData := &pb_common.BlockData{
		Transactions: [][]byte{protoTx},
	}
	metadata := &pb_common.BlockMetadata{
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

func GetBlockDataFromEnvelope(envelope *pb_common.Envelope) (*pb_common.Block, error) {
	payload, err := UnmarshalPayloadFromProto(envelope.Payload)
	if err != nil {
		return nil, err
	}

	block, err := UnmarshalBlockFromProto(payload.Data)
	if err != nil {
		return nil, err
	}

	return block, nil
}

// GetIdentityFromHeader Header에서 Identity 추출
func GetIdentityFromHeader(header *pb_common.Header) (*pb_common.Identity, error) {
	if header.Identity != nil {
		return header.Identity, nil
	}

	return nil, errors.New("no identity found in header")
}

// ValidateEnvelope Envelope 유효성 검증
func ValidateEnvelope(envelope *pb_common.Envelope) error {
	if envelope == nil {
		return errors.New("envelope is nil")
	}

	if len(envelope.Payload) == 0 {
		return errors.New("envelope payload is empty")
	}

	if len(envelope.Signature) == 0 {
		return errors.New("envelope signature is empty")
	}

	return nil
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
