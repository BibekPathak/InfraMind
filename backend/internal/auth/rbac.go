package auth

import (
	"net/http"
)

// Role hierarchy: viewer < operator < admin
var roleRank = map[string]int{
	"viewer":   1,
	"operator": 2,
	"admin":    3,
}

// Permission definitions.
type permission string

const (
	PermRead      permission = "read"
	PermWrite     permission = "write"
	PermApprove   permission = "approve"
	PermConfigure permission = "configure"
	PermAdmin     permission = "admin"
)

// Minimum role required for each permission.
var permissionMinRole = map[permission]string{
	PermRead:      "viewer",
	PermWrite:     "operator",
	PermApprove:   "operator",
	PermConfigure: "operator",
	PermAdmin:     "admin",
}

// RequireRole returns middleware that blocks requests when the authenticated
// user's role is below the minimum required for the given permission.
func RequireRole(perms ...permission) func(http.Handler) http.Handler {
	minRole := "viewer"
	for _, p := range perms {
		if r, ok := permissionMinRole[p]; ok && roleRank[r] > roleRank[minRole] {
			minRole = r
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(RoleKey).(string)
			if role == "" {
				role = "viewer"
			}

			if roleRank[role] < roleRank[minRole] {
				http.Error(w, `{"error":{"code":"FORBIDDEN","message":"insufficient permissions"}}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRoleOnMutation enforces the minimum role only for state-changing
// methods (POST/PUT/PATCH/DELETE); reads (GET) are allowed for all roles.
func RequireRoleOnMutation(perms ...permission) func(http.Handler) http.Handler {
	minRole := "viewer"
	for _, p := range perms {
		if r, ok := permissionMinRole[p]; ok && roleRank[r] > roleRank[minRole] {
			minRole = r
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			role, _ := r.Context().Value(RoleKey).(string)
			if role == "" {
				role = "viewer"
			}

			if roleRank[role] < roleRank[minRole] {
				http.Error(w, `{"error":{"code":"FORBIDDEN","message":"insufficient permissions"}}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin is shorthand for admin-only routes.
func RequireAdmin() func(http.Handler) http.Handler {
	return RequireRole(PermAdmin)
}
