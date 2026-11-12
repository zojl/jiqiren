package repository

import (
	"fmt"
	"time"

	"jiqiren/bot/battletime"
	"jiqiren/bot/database"
	"jiqiren/bot/database/model"
)

func FindAllPlannedTargets() ([]model.PlannedTarget, error) {
	db := database.GetDb()
	targets := []model.PlannedTarget{}
	err := db.Find(&targets).Error
	if (err != nil) {
		return nil, fmt.Errorf("Could not load planned targets")
	}

	return targets, nil
}

func FindPlannedTargetsBySquadId(squadId uint) ([]model.PlannedTarget, error) {
	db := database.GetDb()
	targets := []model.PlannedTarget{}
	err := db.Where("squad_id = ?", squadId).Order("start_date, hour, minute").Find(&targets).Error
	if (err != nil) {
		return nil, fmt.Errorf("Could not load planned targets for squad id: %d", squadId)
	}

	return targets, nil
}

// FindPlannedTargetsByFractionId loads only fraction-scope entries
// (SquadId IS NULL) — squad-scope entries belonging to this fraction are
// reached through FindPlannedTargetsBySquadId instead.
func FindPlannedTargetsByFractionId(fractionId uint) ([]model.PlannedTarget, error) {
	db := database.GetDb()
	targets := []model.PlannedTarget{}
	err := db.Where("fraction_id = ? AND squad_id IS NULL", fractionId).Order("start_date, hour, minute").Find(&targets).Error
	if (err != nil) {
		return nil, fmt.Errorf("Could not load planned targets for fraction id: %d", fractionId)
	}

	return targets, nil
}

func FindOnePlannedTargetById(id uint) (*model.PlannedTarget, error) {
	db := database.GetDb()
	target := model.PlannedTarget{}
	err := db.Where("id = ?", id).First(&target).Error
	if (err != nil) {
		return nil, fmt.Errorf("No planned target with id: %d", id)
	}

	return &target, nil
}

// CreatePlannedTarget pre-marks the entry as already applied today when
// its first due moment has already gone by today — same trick as
// CreateScheduledMessage, so a recurring entry waits for its next day
// instead of firing immediately, while a "once" entry (whose only day is
// today) is deliberately left unmarked so it catches up on the next tick.
func CreatePlannedTarget(target *model.PlannedTarget) error {
	now := time.Now()
	if (target.Type == model.PlannedTargetTypeRecurring && !target.StartDate.After(battletime.Today(now)) && battletime.IsPastToday(target.Hour, target.Minute, now)) {
		today := battletime.Today(now)
		target.LastAppliedDate = &today
	}

	db := database.GetDb()
	return db.Create(target).Error
}

// UpdatePlannedTarget re-derives LastAppliedDate whenever the schedule
// actually changes (type, start date or time of day), with the same logic
// CreatePlannedTarget uses for a fresh entry — cleared when the new
// schedule's moment hasn't happened yet today, set to today when it has
// already gone by (recurring only; "once" is never pre-marked).
func UpdatePlannedTarget(target *model.PlannedTarget, update model.PlannedTarget) error {
	db := database.GetDb()
	updates := map[string]interface{}{
		"target":   update.Target,
		"type":     update.Type,
		"start_date": update.StartDate,
		"end_date": update.EndDate,
		"hour":     update.Hour,
		"minute":   update.Minute,
		"reserve":  update.Reserve,
		"message":  update.Message,
	}

	scheduleChanged := update.Type != target.Type || !update.StartDate.Equal(target.StartDate) ||
		update.Hour != target.Hour || update.Minute != target.Minute

	if (scheduleChanged) {
		now := time.Now()
		if (update.Type == model.PlannedTargetTypeRecurring && !update.StartDate.After(battletime.Today(now)) && battletime.IsPastToday(update.Hour, update.Minute, now)) {
			today := battletime.Today(now)
			updates["last_applied_date"] = today
		} else {
			updates["last_applied_date"] = nil
		}
	}

	return db.Model(target).Updates(updates).Error
}

func DeletePlannedTarget(id uint) error {
	db := database.GetDb()
	return db.Delete(&model.PlannedTarget{}, id).Error
}

func MarkPlannedTargetApplied(target *model.PlannedTarget, date time.Time) error {
	db := database.GetDb()
	return db.Model(target).Updates(map[string]interface{}{
		"last_applied_date": date,
		"last_error":        "",
	}).Error
}

func MarkPlannedTargetFailed(target *model.PlannedTarget, errText string) error {
	db := database.GetDb()
	return db.Model(target).Update("LastError", errText).Error
}
