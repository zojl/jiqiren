package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"time"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"

	"github.com/golang-jwt/jwt/v5"
)

const CookieName = "jiqiren_session"
const sessionLifetime = 7 * 24 * time.Hour
const renewThreshold = 3 * 24 * time.Hour

func signingKey() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
}

func generateSessionId() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); (err != nil) {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

// ensureSessionId returns manager's current session id, lazily generating
// and persisting one on first use (e.g. the very first login after this
// field was introduced) rather than requiring a separate backfill.
func ensureSessionId(manager *model.Manager) (string, error) {
	if (manager.SessionId != nil) {
		return *manager.SessionId, nil
	}

	sessionId, err := generateSessionId()
	if (err != nil) {
		return "", err
	}

	if err := repository.SetManagerSessionId(manager, sessionId); (err != nil) {
		return "", err
	}

	return sessionId, nil
}

// RotateSessionId replaces manager's session id with a freshly generated
// one, invalidating every cookie issued against the old value — the
// "завершить все сессии" action. Callers still need IssueSession
// afterwards to keep the current device logged in under the new id.
func RotateSessionId(manager *model.Manager) (string, error) {
	sessionId, err := generateSessionId()
	if (err != nil) {
		return "", err
	}

	if err := repository.SetManagerSessionId(manager, sessionId); (err != nil) {
		return "", err
	}

	return sessionId, nil
}

func sessionClaims(managerId uint, sessionId string) *SessionClaims {
	now := time.Now()
	return &SessionClaims{
		ManagerId: managerId,
		SessionId: sessionId,
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
	sessionId, err := ensureSessionId(manager)
	if (err != nil) {
		return err
	}

	return setCookie(w, sessionClaims(manager.Id, sessionId))
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

	setCookie(w, sessionClaims(claims.ManagerId, claims.SessionId))
}
