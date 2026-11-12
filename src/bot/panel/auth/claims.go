package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

// SessionClaims is the session cookie's JWT payload — an identity, an
// expiry, and a session id. The identity is a User (anyone with a profile
// may log in), not a Manager. Anything permission-relevant (role,
// fraction, squad) is deliberately not stored here: the manager row is
// looked up fresh from the database on every request (see RequireAuth),
// so a manager's access change or revocation takes effect on their very
// next request instead of staying valid for as long as an old cookie is
// still accepted.
//
// SessionId must match the user's current User.SessionId (checked in
// RequireAuth) — it's how "завершить все сессии" invalidates every
// previously issued cookie at once: regenerating the stored value makes
// every token carrying the old one fail that comparison, regardless of how
// far its expiry still is.
type SessionClaims struct {
	UserId uint `json:"uid"`
	SessionId string `json:"sid"`
	jwt.RegisteredClaims
}
