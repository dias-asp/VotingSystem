package httpapi

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"audit_service/internal/domain"
	"audit_service/internal/service"
)

type Handler struct {
	prover *service.Prover
	chain  ChainVerifier
}

// ChainVerifier is satisfied by *service.ChainVerifier; broken out as an
// interface so handlers compose cleanly in tests.
type ChainVerifier interface {
	Verify(ctx context.Context) (service.ChainStatus, error)
}

func NewHandler(prover *service.Prover, chain ChainVerifier) *Handler {
	return &Handler{prover: prover, chain: chain}
}

type proofStepDTO struct {
	Hash string `json:"hash"`
	Left bool   `json:"left"`
}

type proofDTO struct {
	EventID    string         `json:"event_id"`
	BlockID    int64          `json:"block_id"`
	LeafHash   string         `json:"leaf_hash"`
	MerkleRoot string         `json:"merkle_root"`
	Steps      []proofStepDTO `json:"steps"`
}

func proofToDTO(p domain.MerkleProof) proofDTO {
	steps := make([]proofStepDTO, len(p.Steps))
	for i, s := range p.Steps {
		steps[i] = proofStepDTO{Hash: hex.EncodeToString(s.Hash), Left: s.Left}
	}
	return proofDTO{
		EventID:    p.EventID,
		BlockID:    p.BlockID,
		LeafHash:   hex.EncodeToString(p.LeafHash),
		MerkleRoot: hex.EncodeToString(p.MerkleRoot),
		Steps:      steps,
	}
}

func proofFromDTO(d proofDTO) (domain.MerkleProof, error) {
	leaf, err := hex.DecodeString(d.LeafHash)
	if err != nil {
		return domain.MerkleProof{}, err
	}
	root, err := hex.DecodeString(d.MerkleRoot)
	if err != nil {
		return domain.MerkleProof{}, err
	}
	steps := make([]domain.MerkleProofStep, len(d.Steps))
	for i, s := range d.Steps {
		hash, err := hex.DecodeString(s.Hash)
		if err != nil {
			return domain.MerkleProof{}, err
		}
		steps[i] = domain.MerkleProofStep{Hash: hash, Left: s.Left}
	}
	return domain.MerkleProof{
		EventID:    d.EventID,
		BlockID:    d.BlockID,
		LeafHash:   leaf,
		MerkleRoot: root,
		Steps:      steps,
	}, nil
}

func (h *Handler) Proof(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	if eventID == "" {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "event id required"})
		return
	}
	proof, err := h.prover.Proof(r.Context(), eventID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorBody{Error: "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, proofToDTO(proof))
}

func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	var dto proofDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "invalid body"})
		return
	}
	proof, err := proofFromDTO(dto)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "invalid hex: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Valid bool `json:"valid"`
	}{Valid: h.prover.Verify(proof)})
}

type chainStatusDTO struct {
	Valid       bool   `json:"valid"`
	BlockCount  int    `json:"block_count"`
	LastBlockID int64  `json:"last_block_id"`
	LastHash    string `json:"last_hash"`
	Error       string `json:"error,omitempty"`
}

func (h *Handler) ChainVerify(w http.ResponseWriter, r *http.Request) {
	status, err := h.chain.Verify(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: err.Error()})
		return
	}
	out := chainStatusDTO{
		Valid:       status.Valid,
		BlockCount:  status.BlockCount,
		LastBlockID: status.LastBlockID,
		LastHash:    hex.EncodeToString(status.LastHash),
		Error:       status.Error,
	}
	writeJSON(w, http.StatusOK, out)
}

type errorBody struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
