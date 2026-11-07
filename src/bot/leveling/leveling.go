package leveling

import "math/big"

// LevelMax is the game's level cap — there is no transition from 40 to a
// level "41". Instead, additional progress at 40 earns +1 prestige by
// re-accumulating CumulativeExperience(LevelMax) again starting from
// zero. See ProgressFor.
const LevelMax = 40

// CumulativeExperience returns the total experience needed, accumulated
// from zero, to be at the given level: f(n) = round((1250/3)*(6/5)^n -
// 500). Computed with exact rational arithmetic (not floats) so the
// rounding matches the game's own numbers precisely.
func CumulativeExperience(level uint) uint64 {
	num6Pow := new(big.Int).Exp(big.NewInt(6), big.NewInt(int64(level)), nil)
	den5Pow := new(big.Int).Exp(big.NewInt(5), big.NewInt(int64(level)), nil)

	numerator := new(big.Int).Mul(big.NewInt(1250), num6Pow)
	denominator := new(big.Int).Mul(big.NewInt(3), den5Pow)

	exact := new(big.Rat).Sub(new(big.Rat).SetFrac(numerator, denominator), big.NewRat(500, 1))
	rounded := new(big.Rat).Add(exact, big.NewRat(1, 2))

	return new(big.Int).Div(rounded.Num(), rounded.Denom()).Uint64()
}

// Progress is the experience step a player is currently working through
// — levelling up normally, or (at LevelMax) earning a prestige point.
type Progress struct {
	Remaining uint64
	Percent float64
	Ok bool
}

// ProgressFor reports progress toward a player's next step, given their
// current level and raw (cumulative-since-zero) experience.
//
// Below LevelMax, the step is level→level+1, scored against just that
// step's own size — CumulativeExperience(level+1) - CumulativeExperience(level)
// — rather than the full cumulative total, so banked experience from
// earlier levels doesn't deflate the percentage. At LevelMax the step is
// re-earning the full CumulativeExperience(LevelMax) from zero for +1
// prestige, since there's no level 41.
//
// Ok is false when experience exceeds what the step needs — either a
// stale profile scraped under a different formula, or a level-up/prestige
// the player has already earned but not yet actually taken.
func ProgressFor(level uint, experience uint64) Progress {
	if (level == 0) {
		return Progress{}
	}

	baseline := CumulativeExperience(level)
	target := CumulativeExperience(level + 1)
	if (level >= LevelMax) {
		baseline = 0
		target = CumulativeExperience(LevelMax)
	}

	if (experience > target) {
		return Progress{}
	}

	step := target - baseline
	if (step == 0) {
		return Progress{Remaining: target - experience, Percent: 100, Ok: true}
	}

	return Progress{
		Remaining: target - experience,
		Percent: float64(experience-baseline) / float64(step) * 100,
		Ok: true,
	}
}
