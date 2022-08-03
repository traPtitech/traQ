//go:build tools
// +build tools

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"
)

func main() {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	ecder, _ := x509.MarshalECPrivateKey(priv)
	ecderpub, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)

	pripem, err := os.OpenFile("ec.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatal(err)
	}
	pem.Encode(pripem, &pem.Block{Type: "EC PRIVATE KEY", Bytes: ecder})

	pubpem, err := os.OpenFile("ec_pub.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatal(err)
	}
	pem.Encode(pubpem, &pem.Block{Type: "PUBLIC KEY", Bytes: ecderpub})
}
