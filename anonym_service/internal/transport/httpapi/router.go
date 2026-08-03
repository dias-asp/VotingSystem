package httpapi

import "net/http"

func NewRouter(healthz http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz)
	return mux
}
