package httpapi

import "net/http"

func NewRouter(h *Handler, healthz http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz)
	mux.HandleFunc("POST /auth/register", h.Register)
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("POST /auth/refresh", h.Refresh)
	mux.HandleFunc("POST /auth/logout", h.Logout)
	mux.HandleFunc("GET /.well-known/jwks.json", h.JWKS)
	return mux
}
