package panel

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jiqiren/bot/panel/auth"

	"github.com/go-chi/chi/v5"
)

// Every panel route, hit without a session cookie, must send the visitor
// back to the login page — nothing about the panel leaks to the public.
// Walking the router keeps this true for routes added later, too.
func TestEveryRouteRequiresASession(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	router := NewRouter()

	visited := 0
	err := chi.Walk(router, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if (strings.HasPrefix(route, "/static/")) {
			return nil
		}

		// Fill in path params with something harmless.
		path := strings.NewReplacer("{code}", "AA", "{vk_id}", "1", "{id}", "1").Replace(route)
		request := httptest.NewRequest(method, path, nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		visited++

		if (recorder.Code != http.StatusFound || recorder.Header().Get("Location") != "/") {
			t.Errorf("%s %s without a session: %d → %q, want 302 → /", method, route, recorder.Code, recorder.Header().Get("Location"))
		}

		return nil
	})
	if (err != nil) {
		t.Fatal(err)
	}

	if (visited < 50) {
		t.Fatalf("walked only %d routes — did the walk miss the auth group?", visited)
	}
}

func TestForgedCookieIsRejected(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	router := NewRouter()

	request := httptest.NewRequest("GET", "/profile", nil)
	request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "eyJhbGciOiJIUzI1NiJ9.eyJ1aWQiOjEsInNpZCI6IngifQ.forged"})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if (recorder.Code != http.StatusFound) {
		t.Fatalf("forged cookie: %d", recorder.Code)
	}

	// And the junk cookie is cleared rather than left to retry forever.
	cleared := false
	for _, cookie := range recorder.Result().Cookies() {
		if (cookie.Name == auth.CookieName && cookie.MaxAge < 0) {
			cleared = true
		}
	}
	if (!cleared) {
		t.Fatal("a rejected cookie must be cleared")
	}
}

func TestSecurityHeaders(t *testing.T) {
	router := NewRouter()
	for _, path := range []string{"/profile", "/static/style.css"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest("GET", path, nil))

		headers := recorder.Header()
		if (headers.Get("X-Frame-Options") != "DENY" || headers.Get("X-Content-Type-Options") != "nosniff" || headers.Get("Referrer-Policy") != "same-origin") {
			t.Errorf("%s: headers %v", path, headers)
		}
	}
}
