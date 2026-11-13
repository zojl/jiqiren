package repository

import (
	"errors"
	"fmt"
	"time"

	"jiqiren/bot/database"
	"jiqiren/bot/database/model"

	"gorm.io/gorm"
)

// Every read below skips Hidden profiles (see model.Profile.Hidden) — a
// disowned profile must not leak into squad membership, battle lookups or
// stats. The only exception is the sender's own history listing, which
// opts in via ProfileHistoryFilter.IncludeHidden so it can be restored.
//
// "Latest" always means by message_date (not by id, as gorm's Last()
// would): that's what idx_profiles_user_date can serve directly, where an
// id-ordered lookup makes Postgres walk the whole table backwards.

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
		Where("users.vk_id = ? AND profiles.message_date < ? AND profiles.message_date > ? AND NOT profiles.hidden", vkId, battleMoment, previousBattleMoment).
		Order("profiles.message_date DESC, profiles.id DESC").
		First(&profile).Error

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

	// One LIMIT 1 lookup per member (LATERAL) walks idx_profiles_user_date
	// backwards and stops at the first row, instead of sorting every
	// member's entire history just to keep the newest — which made this
	// query grow with the age of the squad.
	err := db.Raw(`
		SELECT latest.*
		FROM (
			SELECT DISTINCT user_id FROM profiles WHERE team_id = ? AND NOT hidden
			UNION
			SELECT id FROM users WHERE squad_override_id = ?
		) members
		CROSS JOIN LATERAL (
			SELECT * FROM profiles
			WHERE profiles.user_id = members.user_id AND NOT profiles.hidden
			ORDER BY profiles.message_date DESC, profiles.id DESC
			LIMIT 1
		) latest
		ORDER BY latest.user_id
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
		WHERE user_id IN ? AND message_date < ? AND message_date > ? AND NOT hidden
		ORDER BY user_id, message_date DESC
	`, userIds, battleMoment, windowStart).Scan(&profiles).Error

	if (err != nil) {
		return nil, fmt.Errorf("Could not load battle-date profiles")
	}

	return profiles, nil
}

// FindBestAttackProfile returns the profile snapshot, across the player's
// entire history, with the highest ⚔️ attack power — practice*2 +
// cunning*3, the same formula the squad page's battle summary uses.
func FindBestAttackProfile(userId uint) (*model.Profile, error) {
	db := database.GetDb()
	profile := model.Profile{}

	err := db.Where("user_id = ? AND NOT hidden", userId).
		Order("(practice * 2 + cunning * 3) DESC").
		First(&profile).Error

	if (err != nil) {
		return nil, fmt.Errorf("No profiles associated with user id: %d", userId)
	}

	return &profile, nil
}

// FindBestDefenseProfile mirrors FindBestAttackProfile for 🛡️ defense power
// — theory*2 + wisdom*3.
func FindBestDefenseProfile(userId uint) (*model.Profile, error) {
	db := database.GetDb()
	profile := model.Profile{}

	err := db.Where("user_id = ? AND NOT hidden", userId).
		Order("(theory * 2 + wisdom * 3) DESC").
		First(&profile).Error

	if (err != nil) {
		return nil, fmt.Errorf("No profiles associated with user id: %d", userId)
	}

	return &profile, nil
}

func FindLastProfileByVkId(vkId uint) (*model.Profile, error) {
	db := database.GetDb()
	profile := model.Profile{}

	err := db.Model(&profile).Joins("JOIN users ON users.id = profiles.user_id").
		Where("users.vk_id = ? AND NOT profiles.hidden", vkId).
		Order("profiles.message_date DESC, profiles.id DESC").
		First(&profile).Error

	if (err != nil) {
		return nil, fmt.Errorf("No profiles associated with vk id: %d", vkId)
	}

	return &profile, nil
}

func FindOneProfileById(id uint) (*model.Profile, error) {
	db := database.GetDb()
	profile := model.Profile{}
	err := db.Where("id = ?", id).First(&profile).Error
	if (err != nil) {
		return nil, fmt.Errorf("No profile with id: %d", id)
	}

	return &profile, nil
}

// ErrLastVisibleProfile — hiding would leave the user with no visible
// profile at all; at least one must always stay active.
var ErrLastVisibleProfile = errors.New("cannot hide the last visible profile")

// SetProfileHidden hides or restores a profile. Hiding refuses to take the
// user's last visible profile (ErrLastVisibleProfile). The user row is
// locked for the check, so two concurrent hides (two open tabs) can't both
// see "two left" and leave none.
func SetProfileHidden(profile *model.Profile, hidden bool) error {
	db := database.GetDb()
	if (!hidden) {
		return db.Model(profile).Update("hidden", false).Error
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT id FROM users WHERE id = ? FOR UPDATE", profile.UserId).Error; (err != nil) {
			return err
		}

		var visible int64
		if err := tx.Model(&model.Profile{}).Where("user_id = ? AND NOT hidden", profile.UserId).Count(&visible).Error; (err != nil) {
			return err
		}

		if (!CanHideAnotherProfile(visible)) {
			return ErrLastVisibleProfile
		}

		return tx.Model(profile).Update("hidden", true).Error
	})
}

// CanHideAnotherProfile: hiding is allowed only while more than one profile
// is still visible.
func CanHideAnotherProfile(visibleCount int64) bool {
	return visibleCount > 1
}

func CountVisibleProfiles(userId uint) (int64, error) {
	db := database.GetDb()
	var count int64
	err := db.Model(&model.Profile{}).Where("user_id = ? AND NOT hidden", userId).Count(&count).Error

	return count, err
}

// ProfileHistoryFilter narrows a user's own profile history. From is
// inclusive, To exclusive; either may be nil for an open end.
type ProfileHistoryFilter struct {
	From *time.Time
	To *time.Time
	IncludeHidden bool
}

func (filter ProfileHistoryFilter) apply(query *gorm.DB) *gorm.DB {
	if (!filter.IncludeHidden) {
		query = query.Where("NOT hidden")
	}

	if (filter.From != nil) {
		query = query.Where("message_date >= ?", *filter.From)
	}

	if (filter.To != nil) {
		query = query.Where("message_date < ?", *filter.To)
	}

	return query
}

// FindProfileHistory returns one page of the user's profiles, newest
// first, together with the total count matching filter (for paging).
func FindProfileHistory(userId uint, filter ProfileHistoryFilter, offset int, limit int) ([]model.Profile, int64, error) {
	db := database.GetDb()

	var total int64
	countErr := filter.apply(db.Model(&model.Profile{}).Where("user_id = ?", userId)).Count(&total).Error
	if (countErr != nil) {
		return nil, 0, fmt.Errorf("Could not count profile history for user id: %d", userId)
	}

	profiles := []model.Profile{}
	err := filter.apply(db.Model(&model.Profile{}).Where("user_id = ?", userId)).
		Order("message_date DESC, id DESC").
		Offset(offset).
		Limit(limit).
		Find(&profiles).Error
	if (err != nil) {
		return nil, 0, fmt.Errorf("Could not load profile history for user id: %d", userId)
	}

	return profiles, total, nil
}

// FindExperienceSeries returns the user's visible profiles sent since the
// given moment, oldest first, with just the columns the experience chart
// needs.
func FindExperienceSeries(userId uint, since time.Time) ([]model.Profile, error) {
	db := database.GetDb()
	profiles := []model.Profile{}

	err := db.Model(&model.Profile{}).
		Select("id", "message_date", "level", "prestige", "experience").
		Where("user_id = ? AND NOT hidden AND message_date >= ?", userId, since).
		Order("message_date ASC, id ASC").
		Find(&profiles).Error
	if (err != nil) {
		return nil, fmt.Errorf("Could not load experience series for user id: %d", userId)
	}

	return profiles, nil
}

// AttackScoreSql and DefenseScoreSql are ⚔️/🛡️ power in SQL — the same
// formulas as inventory.Item.AttackScore/DefenseScore.
const AttackScoreSql = "(practice * 2 + cunning * 3)"
const DefenseScoreSql = "(theory * 2 + wisdom * 3)"

// FindBestProfileSince returns the user's visible profile sent since the
// given moment with the highest score (AttackScoreSql or DefenseScoreSql).
// The date bound keeps this a short range scan of idx_profiles_user_date
// rather than a sort over the user's whole history.
func FindBestProfileSince(userId uint, since time.Time, scoreSql string) (*model.Profile, error) {
	db := database.GetDb()
	profile := model.Profile{}

	err := db.Where("user_id = ? AND NOT hidden AND message_date >= ?", userId, since).
		Order(scoreSql + " DESC, message_date DESC").
		First(&profile).Error
	if (err != nil) {
		return nil, fmt.Errorf("No profiles for user id %d since %v", userId, since)
	}

	return &profile, nil
}

// FindLastProfileBetween returns the user's latest visible profile sent
// within [from, to].
func FindLastProfileBetween(userId uint, from time.Time, to time.Time) (*model.Profile, error) {
	db := database.GetDb()
	profile := model.Profile{}

	err := db.Where("user_id = ? AND NOT hidden AND message_date >= ? AND message_date <= ?", userId, from, to).
		Order("message_date DESC, id DESC").
		First(&profile).Error
	if (err != nil) {
		return nil, fmt.Errorf("No profiles for user id %d between %v and %v", userId, from, to)
	}

	return &profile, nil
}
