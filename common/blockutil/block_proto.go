package blockutil

import (
	"github.com/ddr4869/minifab/common/msp"
	pb_common "github.com/ddr4869/minifab/proto/common"
)

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

func CreateHeader(signer msp.SigningIdentity, messageType pb_common.MessageType, channelId string) (*pb_common.Header, error) {
	identity := &pb_common.Identity{
		Creator: signer.GetCertificate().Raw,
		MspId:   signer.GetIdentifier().Mspid,
	}
	header := &pb_common.Header{
		Identity:  identity,
		Type:      messageType,
		ChannelId: channelId,
	}
	return header, nil
}
