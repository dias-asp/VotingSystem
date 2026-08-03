package passwords

import "testing"

func TestBcryptHasherRoundTrip(t *testing.T) {
	h := NewBcryptHasher(4)
	hash, err := h.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !h.Verify("correct horse battery staple", hash) {
		t.Fatal("verify true negative")
	}
	if h.Verify("wrong", hash) {
		t.Fatal("verify false positive")
	}
}

func TestBcryptHasherEmpty(t *testing.T) {
	h := NewBcryptHasher(4)
	if _, err := h.Hash(""); err == nil {
		t.Fatal("expected error for empty password")
	}
}
