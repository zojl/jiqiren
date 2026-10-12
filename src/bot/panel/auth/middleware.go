package auth

import (
	"context"
	"net/http"
)

type contextKey string

const claimsContextKey contextKey = "claims"

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := ParseSessionCookie(r)
		if (err != nil) {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		RenewIfNeeded(w, claims)

		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ClaimsFromContext(r *http.Request) *Claims {
	claims, _ := r.Context().Value(claimsContextKey).(*Claims)
	return claims
}
