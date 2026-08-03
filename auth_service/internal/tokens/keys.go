package tokens

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const rsaBits = 2048

func LoadOrGeneratePrivateKey(path string) (*rsa.PrivateKey, error) {
	if path == "" {
		return nil, errors.New("private key path is empty")
	}

	if key, err := loadPrivateKey(path); err == nil {
		return key, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("load private key: %w", err)
	}

	key, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, fmt.Errorf("generate rsa key: %w", err)
	}
	if err := writePrivateKey(path, key); err != nil {
		return nil, fmt.Errorf("write private key: %w", err)
	}
	return key, nil
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid pem")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsaKey, ok := k.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("not an rsa key")
		}
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("unsupported pem type: %s", block.Type)
	}
}

func writePrivateKey(path string, key *rsa.PrivateKey) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der})
	return os.WriteFile(path, pemBytes, 0o600)
}
