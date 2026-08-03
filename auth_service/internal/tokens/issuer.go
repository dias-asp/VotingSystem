package tokens

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"

	"auth_service/internal/domain"
	"auth_service/internal/service"
)

var _ service.TokenIssuer = (*Issuer)(nil)

type Issuer struct {
	privateKey *rsa.PrivateKey
	issuer     string
	keyID      string
}

func NewIssuer(privateKey *rsa.PrivateKey, issuer, keyID string) *Issuer {
	return &Issuer{
		privateKey: privateKey,
		issuer:     issuer,
		keyID:      keyID,
	}
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

type jwtPayload struct {
	Iss   string `json:"iss"`
	Sub   string `json:"sub"`
	Email string `json:"email,omitempty"`
	Role  string `json:"role,omitempty"`
	Jti   string `json:"jti,omitempty"`
	Iat   int64  `json:"iat"`
	Exp   int64  `json:"exp"`
}

func (i *Issuer) Issue(c domain.Claims) (string, error) {
	if i.privateKey == nil {
		return "", errors.New("private key not loaded")
	}
	header, err := json.Marshal(jwtHeader{Alg: "RS256", Typ: "JWT", Kid: i.keyID})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(jwtPayload{
		Iss:   i.issuer,
		Sub:   c.Subject,
		Email: c.Email,
		Role:  c.Role,
		Jti:   c.TokenID,
		Iat:   c.IssuedAt.Unix(),
		Exp:   c.ExpiresAt.Unix(),
	})
	if err != nil {
		return "", err
	}
	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, i.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

type jwk struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

func (i *Issuer) JWKS() ([]byte, error) {
	if i.privateKey == nil {
		return nil, errors.New("private key not loaded")
	}
	pub := &i.privateKey.PublicKey
	set := jwks{Keys: []jwk{{
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		Kid: i.keyID,
		N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}}}
	return json.Marshal(set)
}
