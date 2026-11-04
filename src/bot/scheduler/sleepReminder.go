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

	nextBattle := battletime.Next(now)
	windowStart := nextBattle.Add(-preBattleWindow)

	lines := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		// Unlike missing-profiles' readiness checks, a sleep prediction
		// isn't tied to the last battle — it's only stale once activeSquadProfiles'
		// own silentAfter cutoff says so.
		if (profile.SleepAt == nil) {
			continue
		}

		sleepAt := *profile.SleepAt
		if (sleepAt.Before(windowStart) || sleepAt.After(nextBattle)) {
			continue
		}

		who := mention(vkIdByUserId[profile.UserId], profile.Username)
		when := sleepAt.In(battletime.Moscow).Format("15:04")
		lines = append(lines, fmt.Sprintf("⚠️ %s упадёт спать в %s — меньше 12 часов до битвы", who, when))
	}

	return strings.Join(lines, "\n")
}
