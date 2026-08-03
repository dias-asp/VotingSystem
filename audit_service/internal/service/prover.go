package service

import (
	"context"
	"errors"
	"fmt"

	"audit_service/internal/domain"
)

type Prover struct {
	blocks  BlockRepository
	records RecordRepository
}

func NewProver(blocks BlockRepository, records RecordRepository) *Prover {
	return &Prover{blocks: blocks, records: records}
}

func (p *Prover) Proof(ctx context.Context, eventID string) (domain.MerkleProof, error) {
	rec, err := p.records.GetByEventID(ctx, eventID)
	if err != nil {
		return domain.MerkleProof{}, fmt.Errorf("get record: %w", err)
	}
	if rec.BlockID == nil {
		return domain.MerkleProof{}, ErrNotFound
	}

	block, err := p.blocks.GetByID(ctx, *rec.BlockID)
	if err != nil {
		return domain.MerkleProof{}, fmt.Errorf("get block: %w", err)
	}

	siblings, err := p.blocks.RecordsForBlock(ctx, block.ID)
	if err != nil {
		return domain.MerkleProof{}, fmt.Errorf("records for block: %w", err)
	}

	leaves := make([][]byte, len(siblings))
	index := -1
	for i, s := range siblings {
		leaves[i] = s.Hash
		if s.EventID == eventID {
			index = i
		}
	}
	if index == -1 {
		return domain.MerkleProof{}, errors.New("record missing from its block")
	}

	steps := BuildProof(leaves, index)
	return domain.MerkleProof{
		EventID:    eventID,
		BlockID:    block.ID,
		LeafHash:   rec.Hash,
		Steps:      steps,
		MerkleRoot: block.MerkleRoot,
	}, nil
}

func (p *Prover) Verify(proof domain.MerkleProof) bool {
	return VerifyProof(proof.LeafHash, proof.Steps, proof.MerkleRoot)
}
