package domain

type MerkleProofStep struct {
	Hash []byte
	Left bool
}

type MerkleProof struct {
	EventID    string
	BlockID    int64
	LeafHash   []byte
	Steps      []MerkleProofStep
	MerkleRoot []byte
}
