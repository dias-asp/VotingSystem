package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newSigner(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa key: %v", err)
	}
	return k
}

func issueToken(t *testing.T, key *rsa.PrivateKey, kid, iss, sub string, exp time.Time) string {
	t.Helper()
	header := jwtHeader{Alg: "RS256", Typ: "JWT", Kid: kid}
	hb, _ := json.Marshal(header)
	payload := jwtPayload{Iss: iss, Sub: sub, Iat: time.Now().Unix(), Exp: exp.Unix()}
	pb, _ := json.Marshal(payload)
	signing := base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(pb)
	digest := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func jwksServer(t *testing.T, key *rsa.PrivateKey, kid string) *httptest.Server {
	t.Helper()
	pub := &key.PublicKey
	body, _ := json.Marshal(jwks{Keys: []jwk{{
		Kty: "RSA", Use: "sig", Alg: "RS256", Kid: kid,
		N: base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		E: base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}}})
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
}

func TestVerifier_HappyPath(t *testing.T) {
	key := newSigner(t)
	srv := jwksServer(t, key, "test-1")
	defer srv.Close()

	keys := NewKeySet(srv.URL)
	if err := keys.Refresh(context.Background()); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	v := NewVerifier(keys, "ngvs-auth")
	tok := issueToken(t, key, "test-1", "ngvs-auth", "user-42", time.Now().Add(time.Hour))

	c, err := v.Verify(tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if c.Subject != "user-42" {
		t.Fatalf("subject = %q", c.Subject)
	}
}

func TestVerifier_RejectsExpired(t *testing.T) {
	key := newSigner(t)
	srv := jwksServer(t, key, "test-1")
	defer srv.Close()

	keys := NewKeySet(srv.URL)
	_ = keys.Refresh(context.Background())
	v := NewVerifier(keys, "ngvs-auth")
	tok := issueToken(t, key, "test-1", "ngvs-auth", "user-42", time.Now().Add(-time.Minute))

	_, err := v.Verify(tok)
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("want ErrExpiredToken, got %v", err)
	}
}

func TestVerifier_RejectsWrongIssuer(t *testing.T) {
	key := newSigner(t)
	srv := jwksServer(t, key, "test-1")
	defer srv.Close()

	keys := NewKeySet(srv.URL)
	_ = keys.Refresh(context.Background())
	v := NewVerifier(keys, "ngvs-auth")
	tok := issueToken(t, key, "test-1", "someone-else", "user-42", time.Now().Add(time.Hour))

	if _, err := v.Verify(tok); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("want ErrInvalidToken, got %v", err)
	}
}

func TestVerifier_RejectsUnknownKid(t *testing.T) {
	key := newSigner(t)
	srv := jwksServer(t, key, "test-1")
	defer srv.Close()
	keys := NewKeySet(srv.URL)
	_ = keys.Refresh(context.Background())
	v := NewVerifier(keys, "ngvs-auth")
	tok := issueToken(t, key, "other-kid", "ngvs-auth", "user-42", time.Now().Add(time.Hour))

	if _, err := v.Verify(tok); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("want ErrUnknownKey, got %v", err)
	}
}

func TestVerifier_RejectsTamperedSignature(t *testing.T) {
	key := newSigner(t)
	srv := jwksServer(t, key, "test-1")
	defer srv.Close()
	keys := NewKeySet(srv.URL)
	_ = keys.Refresh(context.Background())
	v := NewVerifier(keys, "ngvs-auth")
	tok := issueToken(t, key, "test-1", "ngvs-auth", "user-42", time.Now().Add(time.Hour))
	parts := strings.Split(tok, ".")
	parts[2] = base64.RawURLEncoding.EncodeToString([]byte("garbage"))
	if _, err := v.Verify(strings.Join(parts, ".")); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("want ErrInvalidToken, got %v", err)
	}
}

func TestMiddleware_PassesClaimsAndRejectsMissing(t *testing.T) {
	key := newSigner(t)
	srv := jwksServer(t, key, "test-1")
	defer srv.Close()
	keys := NewKeySet(srv.URL)
	_ = keys.Refresh(context.Background())
	v := NewVerifier(keys, "ngvs-auth")

	var seenSub string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := FromContext(r.Context())
		if err != nil {
			t.Fatalf("FromContext: %v", err)
		}
		seenSub = c.Subject
		w.WriteHeader(http.StatusOK)
	})
	h := Middleware(v)(next)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	tok := issueToken(t, key, "test-1", "ngvs-auth", "u-1", time.Now().Add(time.Hour))
	r.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK || seenSub != "u-1" {
		t.Fatalf("happy path failed: code=%d sub=%q", rec.Code, seenSub)
	}

	rec = httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, r2)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing header: want 401, got %d", rec.Code)
	}
}
