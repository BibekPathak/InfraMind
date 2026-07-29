package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

type chiRouter interface {
	Get(string, http.HandlerFunc)
	Post(string, http.HandlerFunc)
	Patch(string, http.HandlerFunc)
	Put(string, http.HandlerFunc)
	Delete(string, http.HandlerFunc)
}

var jwtManager *JWTManager

func InitJWT(secret string) {
	jwtManager = NewJWTManager(secret)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if jwtManager == nil {
			ctx := context.WithValue(r.Context(), UserIDKey, "system")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			ctx := context.WithValue(r.Context(), UserIDKey, "anonymous")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			http.Error(w, `{"error":"invalid authorization header"}`, http.StatusUnauthorized)
			return
		}

		claims, err := jwtManager.ValidateToken(tokenStr)
		if err != nil {
			slog.Warn("invalid jwt token", "error", err)
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, RoleKey, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type contextKey string

const UserIDKey contextKey = "user_id"
const RoleKey contextKey = "role"
