package plain

import "testing"

func TestParseLowLevelProfile(t *testing.T) {
	profile := parseProfile(`⚡Ave Huoqiang (AH | 🈵Huǒqiáng)

💻: 9; 💡: 2007/2080
💵: 14; 📈: 11
🔘: 0; 💸: 4929318.138
🔥: 5/5; 🔋: 199%

📡: 14; 💾: 6
📱: 13; 🔎: 5

До 🛌: 41 ч. 33 мин.`)

	if (profile.Status != "⚡" || profile.Username != "Ave Huoqiang" || profile.Squad != "AH" || profile.Fraction != "Huǒqiáng") {
		t.Fatalf("identity: %+v", *profile)
	}

	if (profile.Level != "9" || profile.Experience != "2007" || profile.Money != "14" || profile.Stamina != "199" ||
		profile.Practice != "14" || profile.Theory != "6" || profile.Cunning != "13" || profile.Wisdom != "5" ||
		profile.BeforeSleepHour != "41" || profile.BeforeSleepMinute != "33") {
		t.Fatalf("stats: %+v", *profile)
	}
}
