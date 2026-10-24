package repository

import (
	"fmt"
	"time"

	"jiqiren/bot/database"
	"jiqiren/bot/database/model"
)

func FindLastProfileByVkIdForBattleDate(vkId uint, battleDate *time.Time) (*model.Profile, error) {
	db := database.GetDb()
	profile := model.Profile{}

	battleHourDiff := 15 //battle is at 18:00 msk, means 15:00 utc
	battleDuration := time.Duration(battleHourDiff) * time.Hour
	battleMoment := battleDate.Add(+battleDuration)

	previousBattleHourDiff := 9 //difference between previous battle and battle day midnight: 15+9 = 24
	previousBattleDuration := time.Duration(previousBattleHourDiff) * time.Hour
	previousBattleMoment := battleDate.Add(-previousBattleDuration)

	err := db.Model(&profile).Joins("JOIN users ON users.id = profiles.user_id").
		Where("users.vk_id = ? AND profiles.message_date < ? AND profiles.message_date > ?", vkId, battleMoment, previousBattleMoment).Last(&profile).Error

	if (err != nil) {
		return nil, fmt.Errorf("No profiles associated with vk id: %d for battle at %+v", vkId, battleDate)
	}

	return &profile, nil
}

// FindLatestProfilesForTeam returns, for every user who has ever reported
// being on this team OR is currently manually overridden onto it, that
// user's single most recent profile overall (any team). A user whose
// latest profile now points elsewhere has left — the caller combines
// TeamId with the user's override (Profile.EffectiveSquadId) to tell
// current members from past ones.
func FindLatestProfilesForTeam(teamId uint) ([]model.Profile, error) {
	db := database.GetDb()
	profiles := []model.Profile{}

	err := db.Raw(`
		SELECT DISTINCT ON (profiles.user_id) profiles.*
		FROM profiles
		WHERE profiles.user_id IN (
			SELECT DISTINCT user_id FROM profiles WHERE team_id = ?
			UNION
			SELECT id FROM users WHERE squad_override_id = ?
		)
		ORDER BY profiles.user_id, profiles.message_date DESC
	`, teamId, teamId).Scan(&profiles).Error

	if (err != nil) {
		return nil, fmt.Errorf("Could not load profiles for team id: %d", teamId)
	}

	return profiles, nil
}

// FindProfilesByUserIdsForBattleDate returns, for each user, the single
// profile that was actually in effect for a given past battle — the one
// sent in the ~24h window before it, same as FindLastProfileByVkIdForBattleDate
// and the contest-points calculation use. By the time someone views a
// squad page after the battle, their latest profile overall may already
// be prepping for the next one, so that's not usable here.
func FindProfilesByUserIdsForBattleDate(userIds []uint, battleDate time.Time) ([]model.Profile, error) {
	db := database.GetDb()
	profiles := []model.Profile{}

	battleMoment := battleDate.Add(15 * time.Hour) // battle is at 18:00 msk, means 15:00 utc
	windowStart := battleMoment.Add(-24 * time.Hour)

	err := db.Raw(`
		SELECT DISTINCT ON (user_id) *
		FROM profiles
		WHERE user_id IN ? AND message_date < ? AND message_date > ?
		ORDER BY user_id, message_date DESC
	`, userIds, battleMoment, windowStart).Scan(&profiles).Error

	if (err != nil) {
		return nil, fmt.Errorf("Could not load battle-date profiles")
	}

	return profiles, nil
}

func FindLastProfileByVkId(vkId uint) (*model.Profile, error) {
	db := database.GetDb()
	profile := model.Profile{}

	err := db.Model(&profile).Joins("JOIN users ON users.id = profiles.user_id").
		Where("users.vk_id = ?", vkId).Last(&profile).Error

	if (err != nil) {
		return nil, fmt.Errorf("No profiles associated with vk id: %d", vkId)
	}

	return &profile, nil
}