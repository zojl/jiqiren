package theme

import (
	"testing"

	"jiqiren/bot/database/model"
)

func text(value string) *string {
	return &value
}

func TestResolve(t *testing.T) {
	if mode, scheme := Resolve(nil); (mode != DefaultMode || scheme != DefaultScheme) {
		t.Fatalf("nil user: %s/%s", mode, scheme)
	}

	if mode, scheme := Resolve(&model.User{ThemeMode: text("bogus"), ColorScheme: text("bogus")}); (mode != DefaultMode || scheme != DefaultScheme) {
		t.Fatalf("invalid stored values fall back: %s/%s", mode, scheme)
	}

	validScheme := Schemes[len(Schemes)-1].Id
	if mode, scheme := Resolve(&model.User{ThemeMode: text("day"), ColorScheme: text(validScheme)}); (mode != "day" || scheme != validScheme) {
		t.Fatalf("valid stored values: %s/%s", mode, scheme)
	}
}

func TestValidity(t *testing.T) {
	for _, mode := range Modes {
		if (!IsValidMode(mode.Id)) {
			t.Errorf("listed mode %s is invalid", mode.Id)
		}
	}

	for _, scheme := range Schemes {
		if (!IsValidScheme(scheme.Id)) {
			t.Errorf("listed scheme %s is invalid", scheme.Id)
		}
	}

	if (IsValidMode("") || IsValidScheme("")) {
		t.Fatal("empty is not valid")
	}
}
