package auth

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
	ErrUnknownKey   = errors.New("unknown signing key")
)

type Verifier struct {
	keys   *KeySet
	issuer string
	now    func() time.Time
}

func NewVerifier(keys *KeySet, issuer string) *Verifier {
	return &Verifier{keys: keys, issuer: issuer, now: time.Now}
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

func (v *Verifier) Verify(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var h jwtHeader
	if err := json.Unmarshal(headerBytes, &h); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if h.Alg != "RS256" {
		return Claims{}, ErrInvalidToken
	}
	pub, ok := v.keys.Key(h.Kid)
	if !ok {
		return Claims{}, ErrUnknownKey
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var p jwtPayload
	if err := json.Unmarshal(payloadBytes, &p); err != nil {
		return Claims{}, ErrInvalidToken
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if v.issuer != "" && p.Iss != v.issuer {
		return Claims{}, ErrInvalidToken
	}
	if p.Exp > 0 && v.now().Unix() > p.Exp {
		return Claims{}, ErrExpiredToken
	}
	if p.Sub == "" {
		return Claims{}, ErrInvalidToken
	}
	return Claims{
		Subject:   p.Sub,
		Email:     p.Email,
		Role:      p.Role,
		TokenID:   p.Jti,
		IssuedAt:  p.Iat,
		ExpiresAt: p.Exp,
	}, nil
}
