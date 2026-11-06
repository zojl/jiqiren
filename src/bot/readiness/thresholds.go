package readiness

import "jiqiren/bot/database/model"

// Defaults used whenever a fraction hasn't overridden a threshold. Money
// is a flat cutoff (not level-scaled): Recommended is the top of the
// "отлично" band, Severe is the top of the "приемлемо" band.
const (
	DefaultStaminaSevereThreshold      = 200
	DefaultStaminaRecommendedThreshold = 250
	DefaultMoneyRecommendedThreshold   = 9
	DefaultMoneySevereThreshold        = 100
)

// Thresholds is the single set of readiness cutoffs shared by the
// "профиль принят" reply, the missing-profile cron reminders, and the
// squad table's rating colors — resolved once per fraction via
// ThresholdsFor rather than three separate hardcoded copies.
type Thresholds struct {
	StaminaSevere      uint
	StaminaRecommended uint
	MoneyRecommended   uint
	MoneySevere        uint
}

func ThresholdsFor(fraction *model.Fraction) Thresholds {
	return Thresholds{
		StaminaSevere:      uintOr(fraction.StaminaSevereThreshold, DefaultStaminaSevereThreshold),
		StaminaRecommended: uintOr(fraction.StaminaRecommendedThreshold, DefaultStaminaRecommendedThreshold),
		MoneyRecommended:   uintOr(fraction.MoneyRecommendedThreshold, DefaultMoneyRecommendedThreshold),
		MoneySevere:        uintOr(fraction.MoneySevereThreshold, DefaultMoneySevereThreshold),
	}
}

func uintOr(value *uint, fallback uint) uint {
	if (value == nil) {
		return fallback
	}

	return *value
}

// StaminaRating and MoneyRating are the shared "отлично"/"хорошо"/
// "приемлемо"/"плохо" grading used for the squad table's coloring and
// (via isIdeallyPrepared) the battle summary's ideal-prep count.
func StaminaRating(stamina uint, thresholds Thresholds) string {
	switch {
	case (stamina >= thresholds.StaminaRecommended):
		return "отлично"
	case (stamina >= thresholds.StaminaSevere):
		return "хорошо"
	default:
		return "плохо"
	}
}

func MoneyRating(money uint, thresholds Thresholds) string {
	switch {
	case (money <= thresholds.MoneyRecommended):
		return "отлично"
	case (money <= thresholds.MoneySevere):
		return "приемлемо"
	default:
		return "плохо"
	}
}
