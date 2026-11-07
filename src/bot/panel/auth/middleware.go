package auth

import (
	"context"
	"net/http"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
)

type contextKey string

const managerContextKey contextKey = "manager"

// RequireAuth verifies the session cookie identifies a manager, then loads
// that manager's row fresh from the database on every request — see
// SessionClaims for why permissions are never read from the cookie
// itself. A manager that's since been deleted (or whose cookie fails to
// parse/verify) is treated as not authenticated at all, clearing any
// stale cookie rather than letting it linger.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := ParseSessionCookie(r)
		if (err != nil) {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		manager, err := repository.FindOneManagerById(claims.ManagerId)
		if (err != nil) {
			ClearSession(w)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		RenewIfNeeded(w, claims)

		ctx := context.WithValue(r.Context(), managerContextKey, manager)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ManagerFromContext(r *http.Request) *model.Manager {
	manager, _ := r.Context().Value(managerContextKey).(*model.Manager)
	return manager
}
