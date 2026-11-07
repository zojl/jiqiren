package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

// SessionClaims is the session cookie's JWT payload — just an identity and
// an expiry. Anything permission-relevant (role, fraction, squad) is
// deliberately not stored here: Permissions is built fresh from the
// database on every request (see RequireAuth), so a manager's access
// change or revocation takes effect on their very next request instead of
// staying valid for as long as an old cookie is still accepted.
type SessionClaims struct {
	ManagerId uint `json:"mid"`
	jwt.RegisteredClaims
}
