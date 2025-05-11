package ratelimiter

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func RateLimitMiddleware(rl *RateLimiter, logger *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientID := r.Header.Get("X-Client-ID")
			if clientID == "" {
				clientID = extractClientIP(r) // fallback
			}

			if !rl.AllowRequest(clientID) {
				logger.Warnw("Rate limit exceeded", "client_id", clientID)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)

				resp := ErrorResponse{
					Code:    http.StatusTooManyRequests,
					Message: "Rate limit exceeded",
				}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}


func extractClientIP(r *http.Request) string {
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}

	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	if ips := r.Header.Get("X-Forwarded-For"); ips != "" {
		return strings.TrimSpace(strings.Split(ips, ",")[0])
	}

	return r.RemoteAddr
}
