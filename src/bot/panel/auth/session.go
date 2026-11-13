package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
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

// minSecretLength is what a brute-force of the HS256 key must at least face;
// anything shorter is a typo or a placeholder, not a secret.
const minSecretLength = 32

func signingKey() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
}

// ValidateConfig fails when sessions can't be signed safely — with no
// JWT_SECRET every cookie would be signed with an empty key, i.e. forgeable
// by anyone. Called at startup so a misconfigured deploy refuses to start
// instead of running wide open.
func ValidateConfig() error {
	if (len(signingKey()) == 0) {
		return errors.New("JWT_SECRET is not set — panel sessions would be forgeable")
	}

	if (len(signingKey()) < minSecretLength) {
		return fmt.Errorf("JWT_SECRET is too short: %d bytes, need at least %d", len(signingKey()), minSecretLength)
	}

	return nil
}

func generateSessionId() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); (err != nil) {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

// ensureSessionId returns user's current session id, lazily generating
// and persisting one on first use (e.g. the very first login after this
// field was introduced) rather than requiring a separate backfill.
func ensureSessionId(user *model.User) (string, error) {
	if (user.SessionId != nil) {
		return *user.SessionId, nil
	}

	sessionId, err := generateSessionId()
	if (err != nil) {
		return "", err
	}

	if err := repository.SetUserSessionId(user, sessionId); (err != nil) {
		return "", err
	}

	return sessionId, nil
}

// RotateSessionId replaces user's session id with a freshly generated
// one, invalidating every cookie issued against the old value — the
// "завершить все сессии" action. Callers still need IssueSession
// afterwards to keep the current device logged in under the new id.
func RotateSessionId(user *model.User) (string, error) {
	sessionId, err := generateSessionId()
	if (err != nil) {
		return "", err
	}

	if err := repository.SetUserSessionId(user, sessionId); (err != nil) {
		return "", err
	}

	return sessionId, nil
}

func sessionClaims(userId uint, sessionId string) *SessionClaims {
	now := time.Now()
	return &SessionClaims{
		UserId: userId,
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

func IssueSession(w http.ResponseWriter, user *model.User) error {
	sessionId, err := ensureSessionId(user)
	if (err != nil) {
		return err
	}

	return setCookie(w, sessionClaims(user.Id, sessionId))
}

func ParseSessionCookie(r *http.Request) (*SessionClaims, error) {
	cookie, err := r.Cookie(CookieName)
	if (err != nil) {
		return nil, err
	}

	claims := &SessionClaims{}
	// Pinning the algorithm closes the "alg" confusion class of attacks
	// outright, rather than relying on key-type mismatches to fail them.
	_, err = jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
		return signingKey(), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
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
// to refresh: RequireAuth already re-fetches the user/manager rows on every
// request regardless of how close the cookie is to expiring.
func RenewIfNeeded(w http.ResponseWriter, claims *SessionClaims) {
	if (claims.ExpiresAt == nil) {
		return
	}

	if (time.Until(claims.ExpiresAt.Time) >= renewThreshold) {
		return
	}

	setCookie(w, sessionClaims(claims.UserId, claims.SessionId))
}
