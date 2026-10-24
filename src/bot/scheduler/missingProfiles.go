package scheduler

import (
	"fmt"
	"strings"
	"time"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
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

// silentAfter: a profile this old says nothing useful about readiness for
// the next battle, and pinging someone who's been gone a week is just
// noise — skip them entirely rather than flag them.
const silentAfter = 7 * 24 * time.Hour

func generateMissingProfilesText(squad *model.Team, now time.Time) string {
	profiles, err := repository.FindLatestProfilesForTeam(squad.Id)
	if (err != nil) {
		return ""
	}

	userIds := make([]uint, 0, len(profiles))
	for _, profile := range profiles {
		userIds = append(userIds, profile.UserId)
	}

	users, _ := repository.FindUsersByIds(userIds)
	vkIdByUserId := make(map[uint]uint, len(users))
	for _, user := range users {
		vkIdByUserId[user.Id] = user.VkId
	}

	lines := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		if (profile.TeamId == nil || *profile.TeamId != squad.Id) {
			continue
		}

		if (now.Sub(profile.MessageDate) > silentAfter) {
			continue
		}

		line := missingProfileLine(profile, squad, vkIdByUserId[profile.UserId])
		if (line != "") {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

func missingProfileLine(profile model.Profile, squad *model.Team, vkId uint) string {
	mention := fmt.Sprintf("[id%d|%s]", vkId, profile.Username)

	if (profile.TargetId == nil) {
		return fmt.Sprintf("‼️ %s не выбрал цель", mention)
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
		return fmt.Sprintf("❗ %s %s", mention, strings.Join(severe, " и "))
	}

	mild := make([]string, 0, 2)
	if (profile.Money >= recommendedMoneyThreshold) {
		mild = append(mild, "рекомендуется слить деньги до 0")
	}
	if (profile.Stamina < recommendedStaminaThreshold) {
		mild = append(mild, "рекомендуется пополнить выносливость до 250")
	}

	if (len(mild) > 0) {
		return fmt.Sprintf("⚠️ %s %s", mention, strings.Join(mild, " и "))
	}

	return ""
}
