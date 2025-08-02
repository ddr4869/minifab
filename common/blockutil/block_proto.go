package blockutil

import (
	"encoding/json"

	"github.com/ddr4869/minifab/common/configtx"
	"github.com/ddr4869/minifab/common/logger"
	pb_common "github.com/ddr4869/minifab/proto/common"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

// MarshalBlockToProto 블록을 proto로 직렬화
func MarshalBlockToProto(block *pb_common.Block) ([]byte, error) {
	blockBytes, err := proto.Marshal(block)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal block to proto")
	}
	return blockBytes, nil
}

// UnmarshalBlockFromProto proto에서 블록으로 역직렬화
func UnmarshalBlockFromProto(blockBytes []byte) (*pb_common.Block, error) {
	block := &pb_common.Block{}
	if err := proto.Unmarshal(blockBytes, block); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal block from proto")
	}
	return block, nil
}

// MarshalPayloadToProto Payload를 proto로 직렬화
func MarshalPayloadToProto(payload *pb_common.Payload) ([]byte, error) {
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal payload to proto")
	}
	return payloadBytes, nil
}

// UnmarshalPayloadFromProto proto에서 Payload로 역직렬화
func UnmarshalPayloadFromProto(payloadBytes []byte) (*pb_common.Payload, error) {
	payload := &pb_common.Payload{}
	if err := proto.Unmarshal(payloadBytes, payload); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal payload from proto")
	}
	return payload, nil
}

// MarshalTransactionToProto Transaction을 proto로 직렬화
func MarshalTransactionToProto(tx *pb_common.Transaction) ([]byte, error) {
	txBytes, err := proto.Marshal(tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal transaction to proto")
	}
	return txBytes, nil
}

// UnmarshalTransactionFromProto proto에서 Transaction으로 역직렬화
func UnmarshalTransactionFromProto(txBytes []byte) (*pb_common.Transaction, error) {
	tx := &pb_common.Transaction{}
	if err := proto.Unmarshal(txBytes, tx); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal transaction from proto")
	}
	return tx, nil
}

func MarshalEnvelopeToProto(envelope *pb_common.Envelope) ([]byte, error) {
	envelopeBytes, err := proto.Marshal(envelope)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal envelope to proto")
	}
	return envelopeBytes, nil
}

func UnmarshalEnvelopeFromProto(envelopeBytes []byte) (*pb_common.Envelope, error) {
	envelope := &pb_common.Envelope{}
	if err := proto.Unmarshal(envelopeBytes, envelope); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal envelope from proto")
	}
	return envelope, nil
}

func MarshalIdentityToProto(identity *pb_common.Identity) ([]byte, error) {
	identityBytes, err := proto.Marshal(identity)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal identity to proto")
	}
	return identityBytes, nil
}

func UnmarshalIdentityFromProto(identityBytes []byte) (*pb_common.Identity, error) {
	identity := &pb_common.Identity{}
	if err := proto.Unmarshal(identityBytes, identity); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal identity from proto")
	}
	return identity, nil
}

func CreateEnvelope(payload *pb_common.Payload, signature []byte) (*pb_common.Envelope, error) {
	payloadBytes, err := MarshalPayloadToProto(payload)
	if err != nil {
		return nil, err
	}

	envelope := &pb_common.Envelope{
		Payload:   payloadBytes,
		Signature: signature,
	}

	return envelope, nil
}

func CreatePayload(header *pb_common.Header, data []byte) (*pb_common.Payload, error) {
	payload := &pb_common.Payload{
		Header: header,
		Data:   data,
	}
	return payload, nil
}

func CreateHeader(identity *pb_common.Identity, messageType pb_common.MessageType, channelId string) (*pb_common.Header, error) {
	header := &pb_common.Header{
		Identity:  identity,
		Type:      messageType,
		ChannelId: channelId,
	}

	return header, nil
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

func ExtractAppChannelConfigFromBlock(block *pb_common.Block) (*configtx.ChannelConfig, error) {
	if len(block.Data.Transactions) == 0 {
		return nil, errors.New("no transactions found in block")
	}

	// Transaction proto로 파싱
	protoTx := &pb_common.Transaction{}
	if err := proto.Unmarshal(block.Data.Transactions[0], protoTx); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal transaction")
	}

	var channelConfig configtx.ChannelConfig
	if err := json.Unmarshal(protoTx.Payload, &channelConfig); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal app channel config")
	}

	logger.Infof("✅ Successfully extracted AppChannelConfig from block")
	return &channelConfig, nil
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

// ValidateBlock Block 유효성 검증
func ValidateBlock(block *pb_common.Block) error {
	if block == nil {
		return errors.New("block is nil")
	}

	if block.Header == nil {
		return errors.New("block header is nil")
	}

	if block.Data == nil {
		return errors.New("block data is nil")
	}

	if len(block.Data.Transactions) == 0 {
		return errors.New("block has no transactions")
	}

	return nil
}
