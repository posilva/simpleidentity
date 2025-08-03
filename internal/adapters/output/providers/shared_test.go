package providers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

const (
	testExpectedIssuer   = "expected_issuer"
	testExpectedAudience = "myapp_audience"
	testSubject          = "1234567890"
	testKeyID            = "test_key_id"
	testExpectedNonce    = "test_nonce"
	testEmail            = "player@testmail.com"
)

type TestKeyPairGenerator struct {
	PublicKeyStr  string
	PublicKey     *rsa.PublicKey
	PrivateKeyStr string
	PrivateKey    *rsa.PrivateKey
}

func (g *TestKeyPairGenerator) GenerateRSAKeys() {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	publicKey := &privateKey.PublicKey

	privDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privDER,
	}
	privateKeyPEM := pem.EncodeToMemory(privBlock)
	pubDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		panic(err)
	}
	pubBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubDER,
	}
	publicKeyPEM := pem.EncodeToMemory(pubBlock)
	pub := string(publicKeyPEM)

	g.PrivateKeyStr = string(privateKeyPEM)
	g.PrivateKey = privateKey
	g.PublicKey = publicKey
	g.PublicKeyStr = pub
}
