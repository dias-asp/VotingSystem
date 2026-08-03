package tokens

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"auth_service/internal/domain"
)

func TestIssueVerifyRoundTrip(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	issuer := NewIssuer(key, "test-iss", "kid-1")
	verifier := NewVerifier(&key.PublicKey, "test-iss")

	now := time.Now().UTC().Truncate(time.Second)
	claims := domain.Claims{
		Subject:   "user-123",
		Email:     "u@example.com",
		Role:      domain.RoleAdmin,
		TokenID:   "tok-1",
		IssuedAt:  now,
		ExpiresAt: now.Add(15 * time.Minute),
	}

	token, err := issuer.Issue(claims)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	got, err := verifier.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.Subject != claims.Subject || got.Email != claims.Email || got.TokenID != claims.TokenID || got.Role != claims.Role {
		t.Fatalf("claims mismatch: %+v vs %+v", got, claims)
	}
}

func TestVerifyExpired(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	issuer := NewIssuer(key, "test-iss", "kid-1")
	verifier := NewVerifier(&key.PublicKey, "test-iss")

	past := time.Now().Add(-time.Hour)
	token, err := issuer.Issue(domain.Claims{
		Subject: "u", IssuedAt: past, ExpiresAt: past.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if _, err := verifier.Verify(token); err != ErrExpiredToken {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}

func TestVerifyWrongIssuer(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	issuer := NewIssuer(key, "iss-A", "kid-1")
	verifier := NewVerifier(&key.PublicKey, "iss-B")

	token, _ := issuer.Issue(domain.Claims{
		Subject: "u", IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute),
	})
	if _, err := verifier.Verify(token); err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestJWKS(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	issuer := NewIssuer(key, "iss", "kid-1")
	body, err := issuer.JWKS()
	if err != nil {
		t.Fatalf("jwks: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("empty jwks")
	}
}
