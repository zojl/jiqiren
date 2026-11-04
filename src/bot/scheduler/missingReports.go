package scheduler

import (
	"fmt"
	"strings"
	"time"

	"jiqiren/bot/battletime"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
)

func generateMissingReportsText(squad *model.Team, now time.Time) string {
	profiles, vkIdByUserId, err := activeSquadProfiles(squad, now)
	if (err != nil) {
		return ""
	}

	lastBattleMsk := battletime.Last(now).In(battletime.Moscow)
	lastBattleDate := time.Date(lastBattleMsk.Year(), lastBattleMsk.Month(), lastBattleMsk.Day(), 0, 0, 0, 0, time.UTC)

	userIds := make([]uint, 0, len(profiles))
	for _, profile := range profiles {
		userIds = append(userIds, profile.UserId)
	}

	reports, reportsErr := repository.FindReportsByUserIdsAndDate(userIds, &lastBattleDate)
	if (reportsErr != nil) {
		return ""
	}

	reported := make(map[uint]bool, len(reports))
	for _, report := range reports {
		reported[report.UserId] = true
	}

	lines := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		if (reported[profile.UserId]) {
			continue
		}

		who := mention(vkIdByUserId[profile.UserId], profile.Username)
		lines = append(lines, fmt.Sprintf("‼️ %s не скинул отчёт", who))
	}

	return strings.Join(lines, "\n")
}
