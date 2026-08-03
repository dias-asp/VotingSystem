package service

import (
	"bytes"
	"testing"

	"audit_service/internal/domain"
)

func leafFromString(s string) []byte { return HashLeaf([]byte(s)) }

func TestMerkleRoot_Empty(t *testing.T) {
	if MerkleRoot(nil) != nil {
		t.Fatalf("empty input should return nil")
	}
}

func TestMerkleRoot_Single(t *testing.T) {
	leaf := leafFromString("a")
	root := MerkleRoot([][]byte{leaf})
	if !bytes.Equal(root, leaf) {
		t.Fatalf("single-leaf root must equal the leaf hash")
	}
}

func TestMerkleRoot_TwoLeaves(t *testing.T) {
	a := leafFromString("a")
	b := leafFromString("b")
	root := MerkleRoot([][]byte{a, b})
	want := HashNode(a, b)
	if !bytes.Equal(root, want) {
		t.Fatalf("two-leaf root mismatch")
	}
}

func TestBuildAndVerifyProof_RoundTrip(t *testing.T) {
	cases := []int{1, 2, 3, 4, 5, 7, 8, 16, 17}
	for _, n := range cases {
		leaves := make([][]byte, n)
		for i := 0; i < n; i++ {
			leaves[i] = HashLeaf([]byte{byte(i)})
		}
		root := MerkleRoot(leaves)
		for i := 0; i < n; i++ {
			steps := BuildProof(leaves, i)
			if !VerifyProof(leaves[i], steps, root) {
				t.Fatalf("proof failed for n=%d index=%d", n, i)
			}
		}
	}
}

func TestVerifyProof_RejectsTamperedLeaf(t *testing.T) {
	leaves := [][]byte{leafFromString("a"), leafFromString("b"), leafFromString("c"), leafFromString("d")}
	root := MerkleRoot(leaves)
	steps := BuildProof(leaves, 1)

	if VerifyProof(leafFromString("x"), steps, root) {
		t.Fatalf("verify should fail for wrong leaf")
	}
}

func TestVerifyProof_RejectsTamperedSibling(t *testing.T) {
	leaves := [][]byte{leafFromString("a"), leafFromString("b"), leafFromString("c"), leafFromString("d")}
	root := MerkleRoot(leaves)
	steps := BuildProof(leaves, 1)
	if len(steps) == 0 {
		t.Fatal("expected at least one step")
	}

	tampered := make([]domain.MerkleProofStep, len(steps))
	copy(tampered, steps)
	tampered[0].Hash = append([]byte(nil), tampered[0].Hash...)
	tampered[0].Hash[0] ^= 0xff
	if VerifyProof(leaves[1], tampered, root) {
		t.Fatalf("verify should fail for tampered sibling")
	}
}

func TestBuildProof_OutOfRange(t *testing.T) {
	leaves := [][]byte{leafFromString("a"), leafFromString("b")}
	if got := BuildProof(leaves, -1); got != nil {
		t.Fatalf("expected nil for negative index, got %v", got)
	}
	if got := BuildProof(leaves, 5); got != nil {
		t.Fatalf("expected nil for out-of-range index, got %v", got)
	}
}
