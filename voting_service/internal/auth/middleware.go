package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type errorBody struct {
	Error string `json:"error"`
}

func Middleware(v *Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if h == "" || !strings.HasPrefix(h, "Bearer ") {
				writeUnauthorized(w, "missing bearer token")
				return
			}
			token := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
			if token == "" {
				writeUnauthorized(w, "missing bearer token")
				return
			}
			claims, err := v.Verify(token)
			if err != nil {
				switch {
				case errors.Is(err, ErrExpiredToken):
					writeUnauthorized(w, "token expired")
				case errors.Is(err, ErrUnknownKey):
					writeUnauthorized(w, "unknown signing key")
				default:
					writeUnauthorized(w, "invalid token")
				}
				return
			}
			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Bearer realm="voting"`)
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(errorBody{Error: msg})
}

func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := FromContext(r.Context())
			if err != nil || c.Role != role {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(errorBody{Error: "forbidden"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
