package httpapi

import "net/http"

func NewRouter(h *Handler, healthz http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz)
	mux.HandleFunc("GET /proof/{eventID}", h.Proof)
	mux.HandleFunc("POST /verify", h.Verify)
	mux.HandleFunc("GET /chain/verify", h.ChainVerify)
	return mux
}
