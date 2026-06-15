package hikp2p

import (
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
)

type KeyPair struct {
	Private []byte
	Public  []byte
}

func GenerateECDHKeyPair() (*KeyPair, error) {
	curve := ecdh.P256()
	priv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &KeyPair{
		Private: priv.Bytes(),
		Public:  priv.PublicKey().Bytes(),
	}, nil
}

func DeriveSharedSecret(private, peerPublic []byte) ([]byte, error) {
	curve := ecdh.P256()
	priv, err := curve.NewPrivateKey(private)
	if err != nil {
		return nil, err
	}
	pub, err := curve.NewPublicKey(peerPublic)
	if err != nil {
		return nil, err
	}
	return priv.ECDH(pub)
}

func ChaCha20Encrypt(key, nonce, plaintext []byte) ([]byte, error) {
	// Используем стандартный chacha20 из golang.org/x/crypto/chacha20
	// Для упрощения можно использовать библиотеку, но для демонстрации заглушка
	// В реальном коде нужно импортировать golang.org/x/crypto/chacha20
	panic("implement ChaCha20 using golang.org/x/crypto/chacha20")
}

func HMACSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
