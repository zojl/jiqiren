package readiness

import (
	"testing"

	"jiqiren/bot/database/model"
)

func u(value uint) *uint {
	return &value
}

func TestTargetMatches(t *testing.T) {
	const own = 4
	fractionAttack := &model.Fraction{Id: own, Target: u(2)}
	fractionDefense := &model.Fraction{Id: own, Target: u(model.TargetDefense)}
	fractionNone := &model.Fraction{Id: own}

	cases := []struct {
		name     string
		target   *uint
		squad    *model.Team
		fraction *model.Fraction
		want     bool
	}{
		{"no target chosen", nil, nil, fractionAttack, false},
		{"no squad, matches fraction", u(2), nil, fractionAttack, true},
		{"no squad, wrong fraction target", u(3), nil, fractionAttack, false},
		{"no squad, fraction defends, player defends", u(own), nil, fractionDefense, true},
		{"no squad, fraction defends, player attacks", u(2), nil, fractionDefense, false},
		{"no squad, fraction without directive", u(3), nil, fractionNone, true},
		{"squad without a target — nothing to check", u(3), &model.Team{Id: 1, FractionId: own}, fractionAttack, true},
		{"squad target matches", u(5), &model.Team{Id: 1, FractionId: own, Target: u(5)}, fractionAttack, true},
		{"squad target differs", u(2), &model.Team{Id: 1, FractionId: own, Target: u(5)}, fractionAttack, false},
		{"squad defends", u(own), &model.Team{Id: 1, FractionId: own, Target: u(model.TargetDefense)}, fractionAttack, true},
	}

	for _, c := range cases {
		profile := model.Profile{TargetId: c.target, FractionId: own}
		if got := TargetMatches(profile, c.squad, c.fraction); (got != c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestThresholdsFor(t *testing.T) {
	defaults := ThresholdsFor(&model.Fraction{})
	if (defaults.MoneySevere != DefaultMoneySevereThreshold || defaults.StaminaIdeal != StaminaIdealThreshold ||
		defaults.GearWarn != DefaultGearWarnThreshold || defaults.GearVerySevere != DefaultGearVerySevereThreshold) {
		t.Fatalf("defaults: %+v", defaults)
	}

	overridden := ThresholdsFor(&model.Fraction{MoneySevereThreshold: u(150), GearSevereThreshold: u(30)})
	if (overridden.MoneySevere != 150 || overridden.GearSevere != 30 || overridden.MoneyRecommended != DefaultMoneyRecommendedThreshold) {
		t.Fatalf("overrides: %+v", overridden)
	}
}

func TestRatings(t *testing.T) {
	thresholds := ThresholdsFor(&model.Fraction{})

	staminaCases := map[uint]string{250: "отлично", 240: "отлично", 239: "хорошо", 200: "хорошо", 199: "плохо"}
	for stamina, want := range staminaCases {
		if got := StaminaRating(stamina, thresholds); (got != want) {
			t.Errorf("StaminaRating(%d) = %s, want %s", stamina, got, want)
		}
	}

	moneyCases := map[uint]string{0: "отлично", 9: "отлично", 10: "приемлемо", 100: "приемлемо", 101: "плохо"}
	for money, want := range moneyCases {
		if got := MoneyRating(money, thresholds); (got != want) {
			t.Errorf("MoneyRating(%d) = %s, want %s", money, got, want)
		}
	}
}
