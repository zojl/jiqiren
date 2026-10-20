package auth

import (
	"net/http"

	"jiqiren/bot/audit"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
)

// HandleLoginRedirect is called from the root GET handler when a `login`
// query param is present. It consumes the PIN (one-time use) and issues a
// session cookie before redirecting into the panel.
func HandleLoginRedirect(w http.ResponseWriter, r *http.Request, code string) {
	manager, err := repository.FindOneManagerByLoginCode(code)
	if (err != nil) {
		// A stale/already-used login link is not an error for someone who's
		// already logged in — just let them through on their existing session.
		if (HasValidSession(r)) {
			http.Redirect(w, r, "/panel", http.StatusFound)
			return
		}

		http.Error(w, "Invalid or expired code / 验证码无效或已过期", http.StatusUnauthorized)
		return
	}

	if err := IssueSession(w, manager); err != nil {
		http.Error(w, "Ошибка создания сессии", http.StatusInternalServerError)
		return
	}

	repository.ClearManagerLoginCode(manager)
	audit.Record(manager.FractionId, manager.Id, model.AuditActionLogin, "вход в панель")

	http.Redirect(w, r, "/panel", http.StatusFound)
}
