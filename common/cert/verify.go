package cert

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/asn1"
	"math/big"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/pkg/errors"
)

type ecdsaSignature struct {
	R, S *big.Int
}

// message is not hashed data
func VerifySignature(pubKey crypto.PublicKey, message []byte, signature []byte) (bool, error) {
	switch pubKey := pubKey.(type) {
	case *ecdsa.PublicKey:
		return VerifyECDSA(pubKey, message, signature)
	case *rsa.PublicKey:
		return VerifyRSA(pubKey, message, signature)
	default:
		return false, errors.New("unsupported public key type")
	}
}

func VerifyECDSA(pubKey *ecdsa.PublicKey, message []byte, signature []byte) (bool, error) {
	logger.Infof("verifying ECDSA signature")
	hash := sha256.Sum256(message)

	ecdsaSignature := &ecdsaSignature{}
	asn1.Unmarshal(signature, ecdsaSignature)
	logger.Infof("VerifyECDSA hash: %v", hash)
	return ecdsa.Verify(pubKey, hash[:], ecdsaSignature.R, ecdsaSignature.S), nil
}

func VerifyRSA(pubKey *rsa.PublicKey, message []byte, signature []byte) (bool, error) {
	logger.Infof("verifying RSA signature")
	hash := sha256.Sum256(message)

	err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], signature)
	if err != nil {
		return false, err
	}

	return true, nil
}
