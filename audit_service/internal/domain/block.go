package domain

import "time"

type Block struct {
	ID         int64
	PrevHash   []byte
	MerkleRoot []byte
	Hash       []byte
	CreatedAt  time.Time
	RecordIDs  []string
}
