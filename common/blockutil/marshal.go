package blockutil

import (
	pb_common "github.com/ddr4869/minifab/proto/common"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

func MarshalBlockToProto(block *pb_common.Block) ([]byte, error) {
	blockBytes, err := proto.Marshal(block)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal block to proto")
	}
	return blockBytes, nil
}

func UnmarshalBlockFromProto(blockBytes []byte) (*pb_common.Block, error) {
	block := &pb_common.Block{}
	if err := proto.Unmarshal(blockBytes, block); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal block from proto")
	}
	return block, nil
}

func MarshalPayloadToProto(payload *pb_common.Payload) ([]byte, error) {
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal payload to proto")
	}
	return payloadBytes, nil
}

func UnmarshalPayloadFromProto(payloadBytes []byte) (*pb_common.Payload, error) {
	payload := &pb_common.Payload{}
	if err := proto.Unmarshal(payloadBytes, payload); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal payload from proto")
	}
	return payload, nil
}

func MarshalTransactionToProto(tx *pb_common.Transaction) ([]byte, error) {
	txBytes, err := proto.Marshal(tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal transaction to proto")
	}
	return txBytes, nil
}

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
