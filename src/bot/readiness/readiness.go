package readiness

import "jiqiren/bot/database/model"

// TargetMatches reports whether a player's chosen battle target lines up
// with the directive that applies to them:
//   - in a squad, with the squad having set a target: must match the squad's
//   - not in a squad: must match the fraction's
//   - in a squad that hasn't set a target: nothing to check, always true
//
// Targeting one's own fraction means "defending" — that's how the in-game
// 🔐 icon is parsed, see getTarget in messageHandler/plain/profile.go —
// matching the model.TargetDefense sentinel.
func TargetMatches(profile model.Profile, squad *model.Team, fraction *model.Fraction) bool {
	if (profile.TargetId == nil) {
		return false
	}

	if (squad == nil || squad.Id == 0) {
		return targetMatchesDirective(*profile.TargetId, fraction.Target, fraction.Id)
	}

	if (squad.Target == nil || *squad.Target == model.TargetNone) {
		return true
	}

	return targetMatchesDirective(*profile.TargetId, squad.Target, squad.FractionId)
}

func targetMatchesDirective(targetId uint, directive *uint, ownFractionId uint) bool {
	if (directive == nil || *directive == model.TargetNone) {
		return true
	}

	if (*directive == model.TargetDefense) {
		return targetId == ownFractionId
	}

	return targetId == *directive
}
