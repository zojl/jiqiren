package scheduler

import (
	"fmt"
	"strings"
	"time"

	"jiqiren/bot/battletime"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/readiness"
)

// generateMissingProfilesText mirrors messageHandler/plain/profile.go's
// validateInsertedProfile (same severe-vs-recommendation split, same
// per-fraction readiness.Thresholds), except the target check compares
// against the squad's actual directive rather than just "was any target
// chosen" — that's reserved for the top tier below.
func generateMissingProfilesText(squad *model.Team, now time.Time) string {
	profiles, vkIdByUserId, err := activeSquadProfiles(squad, now)
	if (err != nil) {
		return ""
	}

	fraction, fractionErr := repository.FindOneFractionById(squad.FractionId)
	if (fractionErr != nil) {
		return ""
	}
	thresholds := readiness.ThresholdsFor(fraction)

	lastBattle := battletime.Last(now)

	lines := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		line := missingProfileLine(profile, squad, lastBattle, vkIdByUserId[profile.UserId], thresholds)
		if (line != "") {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

// missingProfileLine treats a profile older than the last battle the same
// as having none: it was prepped for a battle that's already happened and
// says nothing about readiness for the next one.
func missingProfileLine(profile model.Profile, squad *model.Team, lastBattle time.Time, vkId uint, thresholds readiness.Thresholds) string {
	who := mention(vkId, profile.Username)

	if (profile.MessageDate.Before(lastBattle)) {
		return fmt.Sprintf("‼️ %s не обновил профиль перед битвой", who)
	}

	if (profile.TargetId == nil) {
		return fmt.Sprintf("‼️ %s не выбрал цель", who)
	}

	severe := make([]string, 0, 3)
	if (!readiness.TargetMatches(profile, squad)) {
		severe = append(severe, "выбрал неправильную цель")
	}
	if (profile.Money > thresholds.MoneySevere) {
		severe = append(severe, "оставил слишком много денег")
	}
	if (profile.Stamina < thresholds.StaminaSevere) {
		severe = append(severe, "мало выносливости")
	}

	if (len(severe) > 0) {
		return fmt.Sprintf("❗ %s %s", who, strings.Join(severe, " и "))
	}

	mild := make([]string, 0, 2)
	if (profile.Money > thresholds.MoneyRecommended) {
		mild = append(mild, fmt.Sprintf("рекомендуется слить деньги до %d", thresholds.MoneyRecommended))
	}
	if (profile.Stamina < thresholds.StaminaRecommended) {
		mild = append(mild, fmt.Sprintf("рекомендуется пополнить выносливость до %d", thresholds.StaminaRecommended))
	}

	if (len(mild) > 0) {
		return fmt.Sprintf("⚠️ %s %s", who, strings.Join(mild, " и "))
	}

	return ""
}
