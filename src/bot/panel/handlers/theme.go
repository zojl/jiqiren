package handlers

import (
	"net/http"

	"jiqiren/bot/database/repository"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"
	"jiqiren/bot/panel/theme"
)

func init() {
	templates.BuildTheme = buildTheme
}

func buildTheme(r *http.Request) templates.Theme {
	mode, scheme := theme.Resolve(auth.ManagerFromContext(r))
	return templates.Theme{Mode: mode, Scheme: scheme}
}

func SetManagerTheme(w http.ResponseWriter, r *http.Request) {
	manager := auth.ManagerFromContext(r)

	mode := r.FormValue("mode")
	scheme := r.FormValue("scheme")
	if (!theme.IsValidMode(mode) || !theme.IsValidScheme(scheme)) {
		http.Error(w, "Некорректный режим или цветовая схема", http.StatusBadRequest)
		return
	}

	if err := repository.SetManagerTheme(manager, mode, scheme); (err != nil) {
		http.Error(w, "Не удалось сохранить настройки", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/panel/settings", http.StatusFound)
}
