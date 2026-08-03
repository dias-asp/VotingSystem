package tokens

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"auth_service/internal/domain"
	"auth_service/internal/service"
)

var _ service.TokenVerifier = (*Verifier)(nil)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type Verifier struct {
	publicKey *rsa.PublicKey
	issuer    string
	now       func() time.Time
}

func NewVerifier(publicKey *rsa.PublicKey, issuer string) *Verifier {
	return &Verifier{
		publicKey: publicKey,
		issuer:    issuer,
		now:       time.Now,
	}
}

func (v *Verifier) Verify(token string) (domain.Claims, error) {
	if v.publicKey == nil {
		return domain.Claims{}, errors.New("public key not loaded")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return domain.Claims{}, ErrInvalidToken
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return domain.Claims{}, ErrInvalidToken
	}
	var h jwtHeader
	if err := json.Unmarshal(headerBytes, &h); err != nil {
		return domain.Claims{}, ErrInvalidToken
	}
	if h.Alg != "RS256" {
		return domain.Claims{}, ErrInvalidToken
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return domain.Claims{}, ErrInvalidToken
	}
	var p jwtPayload
	if err := json.Unmarshal(payloadBytes, &p); err != nil {
		return domain.Claims{}, ErrInvalidToken
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return domain.Claims{}, ErrInvalidToken
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(v.publicKey, crypto.SHA256, digest[:], sig); err != nil {
		return domain.Claims{}, ErrInvalidToken
	}
	if v.issuer != "" && p.Iss != v.issuer {
		return domain.Claims{}, ErrInvalidToken
	}
	now := v.now().Unix()
	if p.Exp > 0 && now > p.Exp {
		return domain.Claims{}, ErrExpiredToken
	}
	return domain.Claims{
		Subject:   p.Sub,
		Email:     p.Email,
		Role:      p.Role,
		TokenID:   p.Jti,
		IssuedAt:  time.Unix(p.Iat, 0).UTC(),
		ExpiresAt: time.Unix(p.Exp, 0).UTC(),
	}, nil
}
