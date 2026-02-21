package server

import (
	"crypto/subtle"
	"encoding/json"
	"iwaradl/config"
	"net/http"
	"strings"
)

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(config.Cfg.ApiToken)
		if token == "" {
			writeUnauthorized(w)
			return
		}

		provided := bearerToken(r.Header.Get("Authorization"))
		if provided == "" {
			writeUnauthorized(w)
			return
		}

		if subtle.ConstantTimeCompare([]byte(token), []byte(provided)) != 1 {
			writeUnauthorized(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
