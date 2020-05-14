package random

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
)

// GenerateECDSAKey ECDSAによる鍵を生成します
func GenerateECDSAKey() (privRaw []byte, pubRaw []byte) {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	ecder, _ := x509.MarshalECPrivateKey(priv)
	ecderpub, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: ecder}), pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: ecderpub})
}
