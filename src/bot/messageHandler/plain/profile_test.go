package plain

import "testing"

// Anonymized (see CLAUDE.md): nickname, experience and stats are made up.
const lowLevelProfile = `⚡Testoviy Igrok (AH | 🈵Huǒqiáng)

💻: 9; 💡: 1984/2080
💵: 17; 📈: 9
🔘: 0; 💸: 4810233.517
🔥: 5/5; 🔋: 187%

📡: 12; 💾: 7
📱: 11; 🔎: 4

До 🛌: 39 ч. 12 мин.`

func TestParseLowLevelProfile(t *testing.T) {
	profile := parseProfile(lowLevelProfile)

	if (profile.Status != "⚡" || profile.Username != "Testoviy Igrok" || profile.Squad != "AH" || profile.Fraction != "Huǒqiáng") {
		t.Fatalf("identity: %+v", *profile)
	}

	want := map[string]string{
		"Level": profile.Level, "Experience": profile.Experience, "Money": profile.Money, "Stocks": profile.Stocks,
		"Points": profile.Points, "Vkcoin": profile.Vkcoin, "Motivation": profile.Motivation, "MotivationLimit": profile.MotivationLimit,
		"Stamina": profile.Stamina, "Practice": profile.Practice, "Theory": profile.Theory, "Cunning": profile.Cunning, "Wisdom": profile.Wisdom,
		"BeforeSleepHour": profile.BeforeSleepHour, "BeforeSleepMinute": profile.BeforeSleepMinute,
	}
	expected := map[string]string{
		"Level": "9", "Experience": "1984", "Money": "17", "Stocks": "9", "Points": "0", "Vkcoin": "4810233.517",
		"Motivation": "5", "MotivationLimit": "5", "Stamina": "187", "Practice": "12", "Theory": "7", "Cunning": "11", "Wisdom": "4",
		"BeforeSleepHour": "39", "BeforeSleepMinute": "12",
	}
	for field, value := range expected {
		if (want[field] != value) {
			t.Errorf("%s = %q, want %q", field, want[field], value)
		}
	}

	// Resource lines this profile doesn't have stay empty.
	if (profile.Disks != "" || profile.Pages != "" || profile.Chips != "" || profile.Instructions != "") {
		t.Fatalf("absent resources: %+v", *profile)
	}
}

func TestLeadAndWayTypes(t *testing.T) {
	leads := map[string]uint{"🔸": 2, "▫": 1, "": 0, "x": 0}
	for icon, want := range leads {
		if got := getLeadType(icon); (got != want) {
			t.Errorf("getLeadType(%q) = %d, want %d", icon, got, want)
		}
	}

	ways := map[string]uint{"⬛": 1, "⬜": 2, "🔲": 3, "🔳": 4, "": 0, "x": 0}
	for icon, want := range ways {
		if got := getWayType(icon); (got != want) {
			t.Errorf("getWayType(%q) = %d, want %d", icon, got, want)
		}
	}
}
