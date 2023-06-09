package repository

import (
	"time"

	"jiqiren/bot/database"
	"jiqiren/bot/database/model"
)


func CountContestPointsByContestUserAndBattleDate(contest *model.Contest, user *model.User, battleDate *time.Time) int64 {
	db := database.GetDb()
	var contestPointsCount int64

	err := db.Model(&model.ContestPoints{}).
		Where(
			"contest_id = ? AND user_id = ? AND battle_date = ?",
			contest.Id,
			user.Id,
			battleDate,
		).
		Count(&contestPointsCount).Error

	if (err != nil) {
		return 0
	}

	return contestPointsCount
}

func SumPointsByContestAndUser(contest *model.Contest, user *model.User) int64 {
	db := database.GetDb()
	var points int64

	err := db.Model(&model.ContestPoints{}).Where(
		"contest_id = ? AND user_id = ?",
		contest.Id,
		user.Id,
		).Select("SUM(points)").Scan(&points).Error

	if (err != nil) {
		return 0
	}

	return points
}