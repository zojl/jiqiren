package scheduler

import (
	"fmt"
	"strings"
	"time"

	"jiqiren/bot/battletime"
	"jiqiren/bot/database/model"
)

const preBattleWindow = 12 * time.Hour

func generateSleepReminderText(squad *model.Team, now time.Time) string {
	profiles, vkIdByUserId, err := activeSquadProfiles(squad, now)
	if (err != nil) {
		return ""
	}

	lastBattle := battletime.Last(now)
	nextBattle := battletime.Next(now)
	windowStart := nextBattle.Add(-preBattleWindow)

	lines := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		// A profile that predates the last battle is stale — its SleepAt
		// was computed from an old "hours until sleep" reading that has
		// nothing to do with tonight.
		if (profile.MessageDate.Before(lastBattle) || profile.SleepAt == nil) {
			continue
		}

		sleepAt := *profile.SleepAt
		if (sleepAt.Before(windowStart) || sleepAt.After(nextBattle)) {
			continue
		}

		who := mention(vkIdByUserId[profile.UserId], profile.Username)
		when := sleepAt.In(battletime.Moscow).Format("15:04")
		lines = append(lines, fmt.Sprintf("⚠️ %s уйдёт спать в %s — меньше 12 часов до битвы", who, when))
	}

	return strings.Join(lines, "\n")
}
