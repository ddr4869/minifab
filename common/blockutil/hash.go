package blockutil

import (
	"crypto/sha256"
	"encoding/hex"

	pb_common "github.com/ddr4869/minifab/proto/common"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

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
