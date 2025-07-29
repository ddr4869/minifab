package blockutil

import (
	"crypto/sha256"

	pb_common "github.com/ddr4869/minifab/proto/common"
)

func CalculateBlockHash(block *pb_common.Block) []byte {
	if block == nil {
		return nil
	}

	// TODO: 블록 해시 계산 로직 추가
	hash := sha256.New()
	return hash.Sum(block.Header.PreviousHash)
}
