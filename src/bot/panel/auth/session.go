package auth

import (
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

func claimsFromManager(manager *model.Manager) *Claims {
	var accountType *string
	if manager.AccountType != nil {
		value := string(*manager.AccountType)
		accountType = &value
	}

	now := time.Now()
	return &Claims{
		ManagerId: manager.Id,
		FractionId: manager.FractionId,
		SquadId: manager.SquadId,
		AccountType: accountType,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(sessionLifetime)),
		},
	}
}

func setCookie(w http.ResponseWriter, claims *Claims) error {
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
	claims := claimsFromManager(manager)
	return setCookie(w, claims)
}

func ParseSessionCookie(r *http.Request) (*Claims, error) {
	cookie, err := r.Cookie(CookieName)
	if (err != nil) {
		return nil, err
	}

	claims := &Claims{}
	_, err = jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
		return signingKey(), nil
	})
	if (err != nil) {
		return nil, err
	}

	return claims, nil
}

func HasValidSession(r *http.Request) bool {
	_, err := ParseSessionCookie(r)
	return err == nil
}

// RenewIfNeeded extends the session cookie when it's close to expiring.
// The Manager row is refetched only here (not on every request), bounding
// how long a revoked role can keep working via an old cookie.
func RenewIfNeeded(w http.ResponseWriter, claims *Claims) {
	if (claims.ExpiresAt == nil) {
		return
	}

	if (time.Until(claims.ExpiresAt.Time) >= renewThreshold) {
		return
	}

	manager, err := repository.FindOneManagerById(claims.ManagerId)
	if (err != nil) {
		return
	}

	IssueSession(w, manager)
}
