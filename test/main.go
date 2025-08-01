package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"log"
	"math/big"

	"github.com/ddr4869/minifab/peer/core"
)

// newPeer
func newPeer() (*core.Peer, error) {
	peer, err := core.NewPeer("peer0", "Org1MSP", "/Users/mac/go/src/github.com/ddr4869/minifab/ca/Org1/ca-client/peer0", "localhost:7050")
	if err != nil {
		return nil, err
	}
	return peer, nil
}

type ecdsaSignature struct {
	R, S *big.Int
}

func main() {

	peer, err := newPeer()
	if err != nil {
		log.Fatalf("failed to create peer: %v", err)
	}

	message := []byte("testr")
	hash := sha256.Sum256(message)

	sig, err := peer.Client.MSP.GetSigningIdentity().Sign(rand.Reader, hash[:], crypto.SHA256)
	if err != nil {
		log.Fatalf("failed to sign: %v", err)
	}

	cert := peer.Client.MSP.GetSigningIdentity().GetCertificate()
	pubKey := cert.PublicKey

	ecdsaPubKey, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatalf("failed to convert public key to ecdsa public key")
	}

	var esig ecdsaSignature
	_, err = asn1.Unmarshal(sig, &esig)
	if err != nil {
		log.Fatalf("failed to unmarshal ECDSA signature: %v", err)
	}
	ok = ecdsa.Verify(ecdsaPubKey, hash[:], esig.R, esig.S)
	if !ok {
		log.Fatalf("failed to verify signature")
	}

	fmt.Println("signature verified")

	// block, _ := pem.Decode(cert.Raw)
	// if block == nil {
	// 	log.Fatalf("failed to decode PEM block")
	// }

	bc, err := x509.ParseCertificate(cert.Raw)
	if err != nil {
		log.Fatalf("failed to parse certificate: %v", err)
	}

	fmt.Println(bc.Issuer.CommonName)

	// rootCert := peer.Client.MSP.GetRootCertificates()
	// err = rootCert.CheckSignature(rootCert.SignatureAlgorithm, hash[:], sig)
	// if err != nil {
	// 	log.Fatalf("failed to check signature: %v", err)
	// }
	// fmt.Println("signature verified 2")

}
