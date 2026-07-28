package auth

import (
	"context"
	"net/http"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), UserIDKey, "system")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
