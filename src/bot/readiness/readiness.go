package readiness

import "jiqiren/bot/database/model"

// TargetMatches reports whether a player's chosen battle target lines up
// with the squad's directive. Targeting one's own fraction means
// "defending" — that's how the in-game 🔐 icon is parsed, see getTarget in
// messageHandler/plain/profile.go — matching the squad target sentinel
// model.TargetDefense.
func TargetMatches(profile model.Profile, squad *model.Team) bool {
	if (profile.TargetId == nil) {
		return false
	}

	if (squad.Target == nil || *squad.Target == model.TargetNone) {
		return true
	}

	if (*squad.Target == model.TargetDefense) {
		return *profile.TargetId == squad.FractionId
	}

	return *profile.TargetId == *squad.Target
}
