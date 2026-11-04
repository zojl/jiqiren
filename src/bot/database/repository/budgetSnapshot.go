package repository

import (
	"fmt"
	"time"

	"jiqiren/bot/battletime"
	"jiqiren/bot/database"
	"jiqiren/bot/database/model"

	"gorm.io/gorm/clause"
)

// CreateBudgetSnapshot silently ignores a conflict on the (fraction_id,
// captured_at) unique index — a duplicate write for a slot that was already
// recorded (e.g. after a restart lands mid-minute) is not an error, just a
// no-op.
func CreateBudgetSnapshot(snapshot *model.BudgetSnapshot) error {
	db := database.GetDb()
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(snapshot).Error
}

func mskDayBounds(date time.Time) (time.Time, time.Time) {
	dateMsk := date.In(battletime.Moscow)
	start := time.Date(dateMsk.Year(), dateMsk.Month(), dateMsk.Day(), 0, 0, 0, 0, battletime.Moscow)
	return start, start.Add(24 * time.Hour)
}

// FindLatestBudgetSnapshot returns the most recent snapshot (any kind)
// captured on date's MSK calendar day — the seed value shown in the "live
// data" block before it's refreshed (for today) or the last known value for
// a past date (no refresh possible).
func FindLatestBudgetSnapshot(fractionId uint, date time.Time) (*model.BudgetSnapshot, error) {
	dayStart, dayEnd := mskDayBounds(date)

	db := database.GetDb()
	snapshot := model.BudgetSnapshot{}
	err := db.Where("fraction_id = ? AND captured_at >= ? AND captured_at < ?", fractionId, dayStart, dayEnd).
		Order("captured_at DESC").First(&snapshot).Error
	if (err != nil) {
		return nil, fmt.Errorf("no budget snapshot for fraction %d on %s", fractionId, dayStart.Format("2006-01-02"))
	}

	return &snapshot, nil
}

// FindBudgetSnapshotByKindAndDate looks up the single pre_battle or
// post_battle anchor recorded on date's MSK calendar day.
func FindBudgetSnapshotByKindAndDate(fractionId uint, kind model.BudgetSnapshotKind, date time.Time) (*model.BudgetSnapshot, error) {
	dayStart, dayEnd := mskDayBounds(date)

	db := database.GetDb()
	snapshot := model.BudgetSnapshot{}
	err := db.Where("fraction_id = ? AND kind = ? AND captured_at >= ? AND captured_at < ?", fractionId, kind, dayStart, dayEnd).
		Order("captured_at DESC").First(&snapshot).Error
	if (err != nil) {
		return nil, fmt.Errorf("no %s budget snapshot for fraction %d on %s", kind, fractionId, dayStart.Format("2006-01-02"))
	}

	return &snapshot, nil
}

// FindBudgetSnapshotsInRange returns every snapshot (any kind) between from
// and to, ordered by time — the chart's data source.
func FindBudgetSnapshotsInRange(fractionId uint, from time.Time, to time.Time) ([]model.BudgetSnapshot, error) {
	db := database.GetDb()
	snapshots := []model.BudgetSnapshot{}
	err := db.Where("fraction_id = ? AND captured_at >= ? AND captured_at <= ?", fractionId, from, to).
		Order("captured_at ASC").Find(&snapshots).Error
	if (err != nil) {
		return nil, fmt.Errorf("could not load budget snapshots for fraction %d", fractionId)
	}

	return snapshots, nil
}
