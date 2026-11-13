package auth

import (
	"context"
	"net/http"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
)

type contextKey string

const viewerContextKey contextKey = "viewer"

// Viewer is who's behind the current request: always a User, plus the
// Manager row sharing their vk id if they hold a panel role. A plain
// player has a nil Manager, and every role-gated permission is false.
type Viewer struct {
	User *model.User
	Manager *model.Manager
	HasProfile bool
}

// RequireAuth verifies the session cookie identifies a user, then loads
// that user's row — and their manager row, if any — fresh from the
// database on every request; see SessionClaims for why permissions are
// never read from the cookie itself. A user that's since been deleted (or
// whose cookie fails to parse/verify, e.g. a pre-migration manager-id
// cookie) is treated as not authenticated at all, clearing any stale
// cookie rather than letting it linger.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := ParseSessionCookie(r)
		if (err != nil || claims.UserId == 0) {
			ClearSession(w)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		user, err := repository.FindOneUserById(claims.UserId)
		if (err != nil) {
			ClearSession(w)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		if (user.SessionId == nil || claims.SessionId != *user.SessionId) {
			ClearSession(w)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		viewer := &Viewer{
			User: user,
			HasProfile: repository.UserHasProfiles(user.Id),
		}
		if manager, managerErr := repository.FindOneManagerByVkId(user.VkId); (managerErr == nil) {
			viewer.Manager = manager
		}

		RenewIfNeeded(w, claims)

		ctx := context.WithValue(r.Context(), viewerContextKey, viewer)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ViewerFromContext(r *http.Request) *Viewer {
	viewer, _ := r.Context().Value(viewerContextKey).(*Viewer)
	return viewer
}

// UserFromContext is nil only outside RequireAuth.
func UserFromContext(r *http.Request) *model.User {
	if viewer := ViewerFromContext(r); (viewer != nil) {
		return viewer.User
	}

	return nil
}

// ManagerFromContext is nil for a plain player (no panel role).
func ManagerFromContext(r *http.Request) *model.Manager {
	if viewer := ViewerFromContext(r); (viewer != nil) {
		return viewer.Manager
	}

	return nil
}
