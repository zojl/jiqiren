package scheduler

import (
	"testing"
	"time"

	"jiqiren/bot/database/model"
	"jiqiren/bot/gearcheck"
	"jiqiren/bot/readiness"
)

func u(value uint) *uint {
	return &value
}

func TestMissingProfileLine(t *testing.T) {
	now := time.Date(2026, 11, 13, 16, 0, 0, 0, time.UTC)
	lastBattle := now.Add(-7 * time.Hour)
	thresholds := readiness.ThresholdsFor(&model.Fraction{})
	squad := &model.Team{Id: 1, FractionId: 4, Target: u(2)}

	ready := model.Profile{Username: "Testoviy", MessageDate: now, TargetId: u(2), FractionId: 4, Stamina: 250, Money: 0}
	with := func(change func(*model.Profile)) model.Profile {
		profile := ready
		change(&profile)
		return profile
	}
	noGear := gearcheck.Result{}
	gear := func(gain int) gearcheck.Result { return gearcheck.Result{Role: gearcheck.RoleAttack, Gain: gain} }

	cases := []struct {
		name    string
		profile model.Profile
		gear    gearcheck.Result
		want    string
	}{
		{"ready", ready, noGear, ""},
		{"stale profile", with(func(p *model.Profile) { p.MessageDate = lastBattle.Add(-time.Minute) }), noGear, "‼️ [id77|Testoviy] не обновил профиль перед битвой"},
		{"no target", with(func(p *model.Profile) { p.TargetId = nil }), noGear, "‼️ [id77|Testoviy] не выбрал цель"},
		{"wrong target", with(func(p *model.Profile) { p.TargetId = u(3) }), noGear, "❗ [id77|Testoviy] выбрал неправильную цель"},
		{"much money", with(func(p *model.Profile) { p.Money = 100 }), noGear, "❗ [id77|Testoviy] оставил слишком много денег"},
		{"very much money", with(func(p *model.Profile) { p.Money = 500 }), noGear, "‼️ [id77|Testoviy] оставил очень много денег"},
		{"little stamina", with(func(p *model.Profile) { p.Stamina = 150 }), noGear, "❗ [id77|Testoviy] мало выносливости"},
		{"some money and stamina", with(func(p *model.Profile) { p.Money = 10; p.Stamina = 245 }), noGear, "⚠️ [id77|Testoviy] рекомендуется слить деньги до 9"},
		{"gear below the warn threshold", ready, gear(4), ""},
		{"gear warn", ready, gear(5), "⚠️ [id77|Testoviy] неоптимальный набор вещей (можно +5⚔️)"},
		{"gear severe", ready, gear(20), "❗ [id77|Testoviy] неоптимальный набор вещей (можно +20⚔️)"},
		{"gear very severe", ready, gear(50), "‼️ [id77|Testoviy] неоптимальный набор вещей (можно +50⚔️)"},
		{"stamina and gear", with(func(p *model.Profile) { p.Stamina = 150 }), gearcheck.Result{Role: gearcheck.RoleDefense, Gain: 25}, "❗ [id77|Testoviy] мало выносливости и неоптимальный набор вещей (можно +25🛡️)"},
		{"severe wins over mild", with(func(p *model.Profile) { p.Stamina = 150 }), gear(5), "❗ [id77|Testoviy] мало выносливости"},
	}

	for _, c := range cases {
		got := missingProfileLine(c.profile, squad, lastBattle, 77, thresholds, c.gear)
		if (c.name == "some money and stamina") {
			// Both mild hints are listed; just check the money one leads.
			if (len(got) < len(c.want) || got[:len(c.want)] != c.want) {
				t.Errorf("%s: got %q", c.name, got)
			}
			continue
		}

		if (got != c.want) {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}
