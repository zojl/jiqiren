package handlers

import (
	"net/http"

	"jiqiren/bot/panel/auth"
)

// EndAllSessions rotates the manager's session id, which invalidates every
// cookie issued before this request (see auth.RotateSessionId), then issues
// a fresh one so the current device stays logged in.
func EndAllSessions(w http.ResponseWriter, r *http.Request) {
	manager := auth.ManagerFromContext(r)

	if _, err := auth.RotateSessionId(manager); (err != nil) {
		http.Error(w, "Не удалось завершить сессии", http.StatusInternalServerError)
		return
	}

	if err := auth.IssueSession(w, manager); (err != nil) {
		http.Error(w, "Не удалось создать новую сессию", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/panel/settings", http.StatusFound)
}
