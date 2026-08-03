package middleware

import (
	"net/http"
)

const defaultMaxBodyBytes = 1 << 20 // 1 MB

// MaxBody limits request body size and rejects non-JSON bodies on
// content-type-bearing endpoints.
func MaxBody(maxBytes int64) func(http.Handler) http.Handler {
	if maxBytes <= 0 {
		maxBytes = defaultMaxBodyBytes
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}

			// Enforce JSON content type on state-changing requests that carry a body.
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				ct := r.Header.Get("Content-Type")
				if ct != "" && !isJSONContentType(ct) {
					http.Error(w, `{"error":{"code":"UNSUPPORTED_MEDIA_TYPE","message":"Content-Type must be application/json"}}`, http.StatusUnsupportedMediaType)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isJSONContentType(ct string) bool {
	for _, part := range splitComma(ct) {
		trimmed := trimSpace(part)
		if trimmed == "application/json" ||
			hasPrefix(trimmed, "application/json;") ||
			hasPrefix(trimmed, "application/*+json") {
			return true
		}
	}
	return false
}

func splitComma(s string) []string {
	out := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func trimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
