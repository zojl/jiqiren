package readiness

import "jiqiren/bot/database/model"

// StaminaIdealThreshold is fixed, not configurable — it's the game's own
// stamina cap, so there's nothing for a fraction to meaningfully override
// it to. It only gates the battle summary's IdealCount (isIdeallyPrepared),
// never the color rating below.
const StaminaIdealThreshold = 250

// Defaults used whenever a fraction hasn't overridden a threshold.
//
// Both stamina and money have a three-tier message severity (VerySevere/
// Severe/Recommended) that's deliberately coarser than — and doesn't have
// to line up with — the two-tier color rating below: VerySevere only picks
// stronger wording within the same red/"плохо" band Severe already
// defines, same idea as Ideal being stricter than the "отлично" cutoff.
const (
	DefaultStaminaVerySevereThreshold  = 100
	DefaultStaminaSevereThreshold      = 200
	DefaultStaminaRecommendedThreshold = 240
	DefaultMoneyRecommendedThreshold   = 9
	DefaultMoneySevereThreshold        = 100
	DefaultMoneyVerySevereThreshold    = 500
	DefaultGearWarnThreshold           = 5
	DefaultGearSevereThreshold         = 20
	DefaultGearVerySevereThreshold     = 50
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
	MoneyVerySevere    uint
	// Gear* are ⚔️/🛡️ points short of the optimal item set (gearcheck):
	// from Warn on — ⚠️, from Severe — ❗, from VerySevere — ‼️.
	GearWarn       uint
	GearSevere     uint
	GearVerySevere uint
}

func ThresholdsFor(fraction *model.Fraction) Thresholds {
	return Thresholds{
		StaminaVerySevere:  uintOr(fraction.StaminaVerySevereThreshold, DefaultStaminaVerySevereThreshold),
		StaminaSevere:      uintOr(fraction.StaminaSevereThreshold, DefaultStaminaSevereThreshold),
		StaminaRecommended: uintOr(fraction.StaminaRecommendedThreshold, DefaultStaminaRecommendedThreshold),
		StaminaIdeal:       StaminaIdealThreshold,
		MoneyRecommended:   uintOr(fraction.MoneyRecommendedThreshold, DefaultMoneyRecommendedThreshold),
		MoneySevere:        uintOr(fraction.MoneySevereThreshold, DefaultMoneySevereThreshold),
		MoneyVerySevere:    uintOr(fraction.MoneyVerySevereThreshold, DefaultMoneyVerySevereThreshold),
		GearWarn:           uintOr(fraction.GearWarnThreshold, DefaultGearWarnThreshold),
		GearSevere:         uintOr(fraction.GearSevereThreshold, DefaultGearSevereThreshold),
		GearVerySevere:     uintOr(fraction.GearVerySevereThreshold, DefaultGearVerySevereThreshold),
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
// Neither looks at VerySevere (or, for stamina, Ideal) — those only affect
// message wording (see validateInsertedProfile/missingProfileLine), not
// this color rating.
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
