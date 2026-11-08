package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

// SessionClaims is the session cookie's JWT payload — an identity, an
// expiry, and a session id. Anything permission-relevant (role, fraction,
// squad) is deliberately not stored here: Permissions is built fresh from
// the database on every request (see RequireAuth), so a manager's access
// change or revocation takes effect on their very next request instead of
// staying valid for as long as an old cookie is still accepted.
//
// SessionId must match the manager's current Manager.SessionId (checked in
// RequireAuth) — it's how "завершить все сессии" invalidates every
// previously issued cookie at once: regenerating the stored value makes
// every token carrying the old one fail that comparison, regardless of how
// far its expiry still is.
type SessionClaims struct {
	ManagerId uint `json:"mid"`
	SessionId string `json:"sid"`
	jwt.RegisteredClaims
}
