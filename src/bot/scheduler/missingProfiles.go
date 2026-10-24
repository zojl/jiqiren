package scheduler

import (
	"fmt"
	"strings"
	"time"

	"jiqiren/bot/battletime"
	"jiqiren/bot/database/model"
	"jiqiren/bot/readiness"
)

// Thresholds mirror messageHandler/plain/profile.go's validateInsertedProfile
// (same severe-vs-recommendation split), except the target check compares
// against the squad's actual directive rather than just "was any target
// chosen" — that's reserved for the top tier below.
const severeMoneyMultiplier = 10
const severeStaminaThreshold = 200
const recommendedStaminaThreshold = 250
const recommendedMoneyThreshold = 10

func generateMissingProfilesText(squad *model.Team, now time.Time) string {
	profiles, vkIdByUserId, err := activeSquadProfiles(squad, now)
	if (err != nil) {
		return ""
	}

	lastBattle := battletime.Last(now)

	lines := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		line := missingProfileLine(profile, squad, lastBattle, vkIdByUserId[profile.UserId])
		if (line != "") {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

// missingProfileLine treats a profile older than the last battle the same
// as having none: it was prepped for a battle that's already happened and
// says nothing about readiness for the next one.
func missingProfileLine(profile model.Profile, squad *model.Team, lastBattle time.Time, vkId uint) string {
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
	if (profile.Money > profile.Level*severeMoneyMultiplier) {
		severe = append(severe, "оставил слишком много денег")
	}
	if (profile.Stamina < severeStaminaThreshold) {
		severe = append(severe, "мало выносливости")
	}

	if (len(severe) > 0) {
		return fmt.Sprintf("❗ %s %s", who, strings.Join(severe, " и "))
	}

	mild := make([]string, 0, 2)
	if (profile.Money >= recommendedMoneyThreshold) {
		mild = append(mild, "рекомендуется слить деньги до 0")
	}
	if (profile.Stamina < recommendedStaminaThreshold) {
		mild = append(mild, "рекомендуется пополнить выносливость до 250")
	}

	if (len(mild) > 0) {
		return fmt.Sprintf("⚠️ %s %s", who, strings.Join(mild, " и "))
	}

	return ""
}
