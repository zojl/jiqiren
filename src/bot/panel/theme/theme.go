package theme

import "jiqiren/bot/database/model"

const DefaultMode = "night"
const DefaultScheme = "hacker"

type Mode struct {
	Id    string
	Label string
}

type Scheme struct {
	Id     string
	Label  string
	Swatch string // representative accent color, for a small preview dot in the settings UI
}

var Modes = []Mode{
	{Id: "day", Label: "День"},
	{Id: "night", Label: "Ночь"},
	{Id: "auto", Label: "Авто (по настройкам устройства)"},
}

var Schemes = []Scheme{
	{Id: "hacker", Label: "Хакер", Swatch: "#39ff8f"},
	{Id: "solarized", Label: "Solarized", Swatch: "#268bd2"},
	{Id: "gruvbox", Label: "Gruvbox", Swatch: "#fabd2f"},
	{Id: "catppuccin", Label: "Catppuccin", Swatch: "#cba6f7"},
	{Id: "tokyonight", Label: "Tokyo Night", Swatch: "#7aa2f7"},
	{Id: "rosepine", Label: "Rosé Pine", Swatch: "#c4a7e7"},
	{Id: "everforest", Label: "Everforest", Swatch: "#a7c080"},
	{Id: "ayu", Label: "Ayu", Swatch: "#e6b450"},
	{Id: "one", Label: "One", Swatch: "#61afef"},
}

func IsValidMode(id string) bool {
	for _, mode := range Modes {
		if (mode.Id == id) {
			return true
		}
	}

	return false
}

func IsValidScheme(id string) bool {
	for _, scheme := range Schemes {
		if (scheme.Id == id) {
			return true
		}
	}

	return false
}

// Resolve returns the effective mode/scheme for manager, falling back to the
// defaults for a nil manager (e.g. a pre-auth page) or unset preferences.
func Resolve(manager *model.Manager) (mode string, scheme string) {
	mode, scheme = DefaultMode, DefaultScheme

	if (manager == nil) {
		return
	}

	if (manager.ThemeMode != nil && IsValidMode(*manager.ThemeMode)) {
		mode = *manager.ThemeMode
	}

	if (manager.ColorScheme != nil && IsValidScheme(*manager.ColorScheme)) {
		scheme = *manager.ColorScheme
	}

	return
}
