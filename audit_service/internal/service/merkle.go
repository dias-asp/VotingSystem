package service

import (
	"bytes"
	"crypto/sha256"

	"audit_service/internal/domain"
)

// HashLeaf hashes a leaf with a 0x00 domain-separation prefix to make it
// impossible for an internal-node hash to collide with a leaf hash.
func HashLeaf(data []byte) []byte {
	sum := sha256.Sum256(append([]byte{0x00}, data...))
	return sum[:]
}

// HashNode hashes two children with a 0x01 prefix.
func HashNode(left, right []byte) []byte {
	h := sha256.New()
	h.Write([]byte{0x01})
	h.Write(left)
	h.Write(right)
	return h.Sum(nil)
}

// MerkleRoot computes the root over already-hashed leaves. If a level has an
// odd number of nodes, the last node is duplicated.
func MerkleRoot(leaves [][]byte) []byte {
	if len(leaves) == 0 {
		return nil
	}
	level := make([][]byte, len(leaves))
	copy(level, leaves)
	for len(level) > 1 {
		if len(level)%2 == 1 {
			level = append(level, level[len(level)-1])
		}
		next := make([][]byte, 0, len(level)/2)
		for i := 0; i < len(level); i += 2 {
			next = append(next, HashNode(level[i], level[i+1]))
		}
		level = next
	}
	return level[0]
}

// BuildProof returns the inclusion path for the leaf at index. Each step's
// Left flag tells the verifier on which side the sibling sits.
func BuildProof(leaves [][]byte, index int) []domain.MerkleProofStep {
	if index < 0 || index >= len(leaves) {
		return nil
	}
	if len(leaves) == 1 {
		return []domain.MerkleProofStep{}
	}

	level := make([][]byte, len(leaves))
	copy(level, leaves)
	idx := index
	var steps []domain.MerkleProofStep

	for len(level) > 1 {
		if len(level)%2 == 1 {
			level = append(level, level[len(level)-1])
		}
		var sibling []byte
		var siblingLeft bool
		if idx%2 == 0 {
			sibling = level[idx+1]
			siblingLeft = false
		} else {
			sibling = level[idx-1]
			siblingLeft = true
		}
		steps = append(steps, domain.MerkleProofStep{Hash: sibling, Left: siblingLeft})

		next := make([][]byte, 0, len(level)/2)
		for i := 0; i < len(level); i += 2 {
			next = append(next, HashNode(level[i], level[i+1]))
		}
		level = next
		idx /= 2
	}
	return steps
}

// VerifyProof recomputes the root from a leaf + proof and compares against
// the expected root.
func VerifyProof(leaf []byte, steps []domain.MerkleProofStep, root []byte) bool {
	if len(leaf) == 0 || len(root) == 0 {
		return false
	}
	cur := leaf
	for _, s := range steps {
		if s.Left {
			cur = HashNode(s.Hash, cur)
		} else {
			cur = HashNode(cur, s.Hash)
		}
	}
	return bytes.Equal(cur, root)
}
