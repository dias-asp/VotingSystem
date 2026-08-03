package httpapi

import (
	"net/http"

	"voting_service/internal/auth"
	"voting_service/internal/domain"
)

type Middleware func(http.Handler) http.Handler

func NewRouter(h *Handler, authMW Middleware, healthz http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz)

	mux.Handle("GET /polls", authMW(http.HandlerFunc(h.ListPolls)))
	mux.Handle("GET /polls/{id}", authMW(http.HandlerFunc(h.GetPoll)))
	mux.Handle("GET /polls/{id}/results", authMW(http.HandlerFunc(h.Results)))
	mux.Handle("GET /polls/{id}/me", authMW(http.HandlerFunc(h.MyVote)))

	mux.Handle("POST /vote", authMW(http.HandlerFunc(h.CastVote)))
	mux.Handle("POST /revote", authMW(http.HandlerFunc(h.Revote)))
	mux.Handle("POST /cancel-vote", authMW(http.HandlerFunc(h.CancelVote)))

	adminMW := func(next http.Handler) http.Handler {
		return authMW(auth.RequireRole(domain.RoleAdmin)(next))
	}
	mux.Handle("POST /admin/polls", adminMW(http.HandlerFunc(h.CreatePoll)))
	mux.Handle("POST /admin/polls/{id}/candidates", adminMW(http.HandlerFunc(h.AddCandidate)))
	mux.Handle("PATCH /admin/polls/{id}/status", adminMW(http.HandlerFunc(h.SetPollStatus)))

	return mux
}
