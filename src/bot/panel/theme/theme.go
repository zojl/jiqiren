package theme

import "jiqiren/bot/database/model"

const DefaultMode = "night"
const DefaultScheme = "hacker"

type Mode struct {
	Id    string
	Label string
}

// Preview is a handful of representative colors for one mode (day or
// night) of a scheme, rendered as a small gradient strip in the settings
// UI so a scheme's two variants can be told apart at a glance.
type Preview struct {
	Bg     string
	Accent string
	Warn   string
	Danger string
}

type Scheme struct {
	Id    string
	Label string
	Day   Preview
	Night Preview
}

var Modes = []Mode{
	{Id: "day", Label: "День"},
	{Id: "night", Label: "Ночь"},
	{Id: "auto", Label: "Авто (по настройкам устройства)"},
}

var Schemes = []Scheme{
	{Id: "hacker", Label: "Хакер",
		Day:   Preview{Bg: "#f3faf0", Accent: "#0f8a46", Warn: "#a9791a", Danger: "#c43d3d"},
		Night: Preview{Bg: "#0a0d0a", Accent: "#39ff8f", Warn: "#e8c547", Danger: "#ff5f5f"}},
	{Id: "solarized", Label: "Solarized",
		Day:   Preview{Bg: "#fdf6e3", Accent: "#268bd2", Warn: "#b58900", Danger: "#dc322f"},
		Night: Preview{Bg: "#002b36", Accent: "#268bd2", Warn: "#b58900", Danger: "#dc322f"}},
	{Id: "gruvbox", Label: "Gruvbox",
		Day:   Preview{Bg: "#fbf1c7", Accent: "#d79921", Warn: "#d65d0e", Danger: "#cc241d"},
		Night: Preview{Bg: "#282828", Accent: "#fabd2f", Warn: "#fe8019", Danger: "#fb4934"}},
	{Id: "catppuccin", Label: "Catppuccin",
		Day:   Preview{Bg: "#eff1f5", Accent: "#8839ef", Warn: "#df8e1d", Danger: "#d20f39"},
		Night: Preview{Bg: "#1e1e2e", Accent: "#cba6f7", Warn: "#f9e2af", Danger: "#f38ba8"}},
	{Id: "tokyonight", Label: "Tokyo Night",
		Day:   Preview{Bg: "#e6e7ed", Accent: "#2959aa", Warn: "#8f5e15", Danger: "#8c4351"},
		Night: Preview{Bg: "#1a1b26", Accent: "#7aa2f7", Warn: "#e0af68", Danger: "#f7768e"}},
	{Id: "rosepine", Label: "Rosé Pine",
		Day:   Preview{Bg: "#faf4ed", Accent: "#907aa9", Warn: "#ea9d34", Danger: "#b4637a"},
		Night: Preview{Bg: "#191724", Accent: "#c4a7e7", Warn: "#f6c177", Danger: "#eb6f92"}},
	{Id: "everforest", Label: "Everforest",
		Day:   Preview{Bg: "#fdf6e3", Accent: "#8da101", Warn: "#dfa000", Danger: "#f85552"},
		Night: Preview{Bg: "#2d353b", Accent: "#a7c080", Warn: "#dbbc7f", Danger: "#e67e80"}},
	{Id: "ayu", Label: "Ayu",
		Day:   Preview{Bg: "#f8f9fa", Accent: "#f29718", Warn: "#eba400", Danger: "#e65050"},
		Night: Preview{Bg: "#0d1017", Accent: "#e6b450", Warn: "#ffb454", Danger: "#d95757"}},
	{Id: "one", Label: "One",
		Day:   Preview{Bg: "#fafafa", Accent: "#4078f2", Warn: "#c18401", Danger: "#e45649"},
		Night: Preview{Bg: "#282c34", Accent: "#61afef", Warn: "#e5c07b", Danger: "#e06c75"}},
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
