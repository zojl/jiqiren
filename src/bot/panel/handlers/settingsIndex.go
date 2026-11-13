package handlers

import (
	"net/http"

	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"
	"jiqiren/bot/panel/theme"
)

type settingsIndexData struct {
	ThemeMode   string
	ColorScheme string
	Modes       []theme.Mode
	Schemes     []theme.Scheme
}

func SettingsIndex(w http.ResponseWriter, r *http.Request) {
	mode, scheme := theme.Resolve(auth.UserFromContext(r))

	templates.Render(w, r, "settings_index", settingsIndexData{
		ThemeMode:   mode,
		ColorScheme: scheme,
		Modes:       theme.Modes,
		Schemes:     theme.Schemes,
	})
}
