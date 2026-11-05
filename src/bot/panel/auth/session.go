package auth

import (
	"net/http"
	"os"
	"time"

	"jiqiren/bot/database/model"

	"github.com/golang-jwt/jwt/v5"
)

const CookieName = "jiqiren_session"
const sessionLifetime = 7 * 24 * time.Hour
const renewThreshold = 3 * 24 * time.Hour

func signingKey() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
}

func sessionClaimsFromManagerId(managerId uint) *SessionClaims {
	now := time.Now()
	return &SessionClaims{
		ManagerId: managerId,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(sessionLifetime)),
		},
	}
}

func setCookie(w http.ResponseWriter, claims *SessionClaims) error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(signingKey())
	if (err != nil) {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name: CookieName,
		Value: signed,
		Path: "/",
		HttpOnly: true,
		Secure: true,
		SameSite: http.SameSiteLaxMode,
		Expires: claims.ExpiresAt.Time,
	})

	return nil
}

func IssueSession(w http.ResponseWriter, manager *model.Manager) error {
	return setCookie(w, sessionClaimsFromManagerId(manager.Id))
}

func ParseSessionCookie(r *http.Request) (*SessionClaims, error) {
	cookie, err := r.Cookie(CookieName)
	if (err != nil) {
		return nil, err
	}

	claims := &SessionClaims{}
	_, err = jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
		return signingKey(), nil
	})
	if (err != nil) {
		return nil, err
	}

	return claims, nil
}

func ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: CookieName,
		Value: "",
		Path: "/",
		HttpOnly: true,
		Secure: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge: -1,
	})
}

func HasValidSession(r *http.Request) bool {
	_, err := ParseSessionCookie(r)
	return err == nil
}

// RenewIfNeeded extends the session cookie when it's close to expiring —
// just a fresh expiry for the same identity. It carries no permission data
// to refresh: RequireAuth already re-fetches the manager row on every
// request regardless of how close the cookie is to expiring.
func RenewIfNeeded(w http.ResponseWriter, claims *SessionClaims) {
	if (claims.ExpiresAt == nil) {
		return
	}

	if (time.Until(claims.ExpiresAt.Time) >= renewThreshold) {
		return
	}

	setCookie(w, sessionClaimsFromManagerId(claims.ManagerId))
}
