package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"voting_service/internal/auth"
	"voting_service/internal/domain"
	"voting_service/internal/service"
)

type Handler struct {
	voting *service.Voting
	polls  *service.Polls
}

func NewHandler(voting *service.Voting, polls *service.Polls) *Handler {
	return &Handler{voting: voting, polls: polls}
}

type voteRequest struct {
	PollID      string `json:"poll_id"`
	CandidateID string `json:"candidate_id"`
}

type cancelRequest struct {
	PollID string `json:"poll_id"`
}

type resultsResponse struct {
	PollID  string         `json:"poll_id"`
	Results map[string]int `json:"results"`
}

type errorBody struct {
	Error string `json:"error"`
}

type pollDTO struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

func toPollDTO(p domain.Poll) pollDTO {
	return pollDTO{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
		Status:      string(p.Status),
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type candidateDTO struct {
	ID     string `json:"id"`
	PollID string `json:"poll_id"`
	Name   string `json:"name"`
}

func toCandidateDTO(c domain.Candidate) candidateDTO {
	return candidateDTO{ID: c.ID, PollID: c.PollID, Name: c.Name}
}

func (h *Handler) CastVote(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var req voteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := h.voting.Cast(r.Context(), domain.VoteRequest{PollID: req.PollID, UserID: userID, CandidateID: req.CandidateID}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) Revote(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var req voteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := h.voting.Change(r.Context(), domain.VoteRequest{PollID: req.PollID, UserID: userID, CandidateID: req.CandidateID}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) CancelVote(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var req cancelRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := h.voting.Cancel(r.Context(), req.PollID, userID); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) Results(w http.ResponseWriter, r *http.Request) {
	pollID := r.PathValue("id")
	results, err := h.voting.Results(r.Context(), pollID)
	if err != nil {
		writeError(w, err)
		return
	}
	if results == nil {
		results = map[string]int{}
	}
	writeJSON(w, http.StatusOK, resultsResponse{PollID: pollID, Results: results})
}

type myVoteResponse struct {
	PollID      string `json:"poll_id"`
	CandidateID string `json:"candidate_id,omitempty"`
	Type        string `json:"type,omitempty"`
	HasVote     bool   `json:"has_vote"`
}

func (h *Handler) MyVote(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	pollID := r.PathValue("id")
	v, err := h.voting.MyVote(r.Context(), pollID, userID)
	if err != nil {
		if errors.Is(err, service.ErrNoVote) {
			writeJSON(w, http.StatusOK, myVoteResponse{PollID: pollID, HasVote: false})
			return
		}
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, myVoteResponse{
		PollID:      v.PollID,
		CandidateID: v.CandidateID,
		Type:        string(v.Type),
		HasVote:     true,
	})
}

func (h *Handler) ListPolls(w http.ResponseWriter, r *http.Request) {
	polls, err := h.polls.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]pollDTO, 0, len(polls))
	for _, p := range polls {
		out = append(out, toPollDTO(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"polls": out})
}

func (h *Handler) GetPoll(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := h.polls.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	cands, err := h.polls.Candidates(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	candDTOs := make([]candidateDTO, 0, len(cands))
	for _, c := range cands {
		candDTOs = append(candDTOs, toCandidateDTO(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"poll":       toPollDTO(p),
		"candidates": candDTOs,
	})
}

type createPollRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (h *Handler) CreatePoll(w http.ResponseWriter, r *http.Request) {
	var req createPollRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	p, err := h.polls.Create(r.Context(), service.CreatePollInput{Title: req.Title, Description: req.Description})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toPollDTO(p))
}

type addCandidateRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *Handler) AddCandidate(w http.ResponseWriter, r *http.Request) {
	pollID := r.PathValue("id")
	var req addCandidateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	c, err := h.polls.AddCandidate(r.Context(), service.AddCandidateInput{PollID: pollID, ID: req.ID, Name: req.Name})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toCandidateDTO(c))
}

type setStatusRequest struct {
	Status string `json:"status"`
}

func (h *Handler) SetPollStatus(w http.ResponseWriter, r *http.Request) {
	pollID := r.PathValue("id")
	var req setStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := h.polls.SetStatus(r.Context(), pollID, domain.PollStatus(req.Status)); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func requireUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	c, err := auth.FromContext(r.Context())
	if err != nil || c.Subject == "" {
		writeJSON(w, http.StatusUnauthorized, errorBody{Error: "unauthenticated"})
		return "", false
	}
	return c.Subject, true
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return service.ErrInvalidRequest
		}
		return service.ErrInvalidRequest
	}
	return nil
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidRequest):
		writeJSON(w, http.StatusBadRequest, errorBody{Error: err.Error()})
	case errors.Is(err, service.ErrPollNotFound),
		errors.Is(err, service.ErrCandidateNotFound):
		writeJSON(w, http.StatusNotFound, errorBody{Error: err.Error()})
	case errors.Is(err, service.ErrPollExists),
		errors.Is(err, service.ErrCandidateExists),
		errors.Is(err, service.ErrAlreadyVoted),
		errors.Is(err, service.ErrNoActiveVote),
		errors.Is(err, service.ErrCannotChange),
		errors.Is(err, service.ErrPollNotActive):
		writeJSON(w, http.StatusConflict, errorBody{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: "internal error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
