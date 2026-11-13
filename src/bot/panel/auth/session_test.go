package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jiqiren/bot/database/model"

	"github.com/golang-jwt/jwt/v5"
)

func sessionUser() *model.User {
	sessionId := "session-a"
	return &model.User{Id: 42, SessionId: &sessionId}
}

func cookieFrom(t *testing.T, recorder *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range recorder.Result().Cookies() {
		if (cookie.Name == CookieName) {
			return cookie
		}
	}

	t.Fatal("no session cookie set")
	return nil
}

func requestWith(cookie *http.Cookie) *http.Request {
	request := httptest.NewRequest("GET", "/panel", nil)
	if (cookie != nil) {
		request.AddCookie(cookie)
	}

	return request
}

func TestIssueAndParseSession(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	recorder := httptest.NewRecorder()
	if err := IssueSession(recorder, sessionUser()); (err != nil) {
		t.Fatal(err)
	}

	cookie := cookieFrom(t, recorder)
	if (!cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/") {
		t.Fatalf("cookie attributes: %+v", cookie)
	}

	claims, err := ParseSessionCookie(requestWith(cookie))
	if (err != nil || claims.UserId != 42 || claims.SessionId != "session-a") {
		t.Fatalf("claims: %+v, %v", claims, err)
	}

	if (time.Until(claims.ExpiresAt.Time) < sessionLifetime-time.Minute) {
		t.Fatalf("expiry: %v", claims.ExpiresAt)
	}

	if (!HasValidSession(requestWith(cookie)) || HasValidSession(requestWith(nil))) {
		t.Fatal("HasValidSession")
	}
}

func TestRejectsForgedSessions(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	recorder := httptest.NewRecorder()
	IssueSession(recorder, sessionUser())
	cookie := cookieFrom(t, recorder)

	tampered := *cookie
	parts := strings.Split(tampered.Value, ".")
	parts[1] = parts[1][:len(parts[1])-2] + "AA"
	tampered.Value = strings.Join(parts, ".")
	if _, err := ParseSessionCookie(requestWith(&tampered)); (err == nil) {
		t.Fatal("a tampered payload must be rejected")
	}

	t.Setenv("JWT_SECRET", "another-secret")
	if _, err := ParseSessionCookie(requestWith(cookie)); (err == nil) {
		t.Fatal("a token signed with another secret must be rejected")
	}
}

func TestRejectsExpiredSession(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	claims := sessionClaims(42, "session-a")
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
	signed, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(signingKey())

	if _, err := ParseSessionCookie(requestWith(&http.Cookie{Name: CookieName, Value: signed})); (err == nil) {
		t.Fatal("an expired token must be rejected")
	}
}

func TestRenewIfNeeded(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	fresh := sessionClaims(42, "session-a")
	recorder := httptest.NewRecorder()
	RenewIfNeeded(recorder, fresh)
	if (len(recorder.Result().Cookies()) != 0) {
		t.Fatal("a fresh session must not be re-issued")
	}

	expiring := sessionClaims(42, "session-a")
	expiring.ExpiresAt = jwt.NewNumericDate(time.Now().Add(renewThreshold - time.Hour))
	recorder = httptest.NewRecorder()
	RenewIfNeeded(recorder, expiring)

	renewed, err := ParseSessionCookie(requestWith(cookieFrom(t, recorder)))
	if (err != nil || renewed.UserId != 42 || renewed.SessionId != "session-a" || time.Until(renewed.ExpiresAt.Time) < sessionLifetime-time.Minute) {
		t.Fatalf("renewed: %+v, %v", renewed, err)
	}
}

func TestClearSession(t *testing.T) {
	recorder := httptest.NewRecorder()
	ClearSession(recorder)
	if cookie := cookieFrom(t, recorder); (cookie.MaxAge >= 0 || cookie.Value != "") {
		t.Fatalf("cleared cookie: %+v", cookie)
	}
}
