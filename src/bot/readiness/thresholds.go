package readiness

import "jiqiren/bot/database/model"

// Defaults used whenever a fraction hasn't overridden a threshold. Money
// is a flat cutoff (not level-scaled): Recommended is the top of the
// "отлично" band, Severe is the top of the "приемлемо" band.
//
// Stamina has four tiers, not a strict ladder of color bands: VerySevere
// and Severe only pick wording ("очень мало" vs "mало") within the same
// red/"плохо" band below Recommended — the green/"отлично" cutoff for
// rating and coloring purposes. Ideal is stricter still and *not* part of
// the color rating at all — it only gates the battle summary's IdealCount
// (isIdeallyPrepared), so e.g. 242 reads as "отлично" (green, no nudge)
// but still doesn't count as fully topped-up for that tile.
const (
	DefaultStaminaVerySevereThreshold  = 100
	DefaultStaminaSevereThreshold      = 200
	DefaultStaminaRecommendedThreshold = 240
	DefaultStaminaIdealThreshold       = 250
	DefaultMoneyRecommendedThreshold   = 9
	DefaultMoneySevereThreshold        = 100
)

// Thresholds is the single set of readiness cutoffs shared by the
// "профиль принят" reply, the missing-profile cron reminders, and the
// squad table's rating colors — resolved once per fraction via
// ThresholdsFor rather than three separate hardcoded copies.
type Thresholds struct {
	StaminaVerySevere  uint
	StaminaSevere      uint
	StaminaRecommended uint
	StaminaIdeal       uint
	MoneyRecommended   uint
	MoneySevere        uint
}

func ThresholdsFor(fraction *model.Fraction) Thresholds {
	return Thresholds{
		StaminaVerySevere:  uintOr(fraction.StaminaVerySevereThreshold, DefaultStaminaVerySevereThreshold),
		StaminaSevere:      uintOr(fraction.StaminaSevereThreshold, DefaultStaminaSevereThreshold),
		StaminaRecommended: uintOr(fraction.StaminaRecommendedThreshold, DefaultStaminaRecommendedThreshold),
		StaminaIdeal:       uintOr(fraction.StaminaIdealThreshold, DefaultStaminaIdealThreshold),
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
//
// StaminaRating deliberately doesn't look at VerySevere/Ideal — those two
// only affect message wording (see validateInsertedProfile/
// missingProfileLine), not this three-band color rating.
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
