package repository

import (
	"time"

	"jiqiren/bot/database"
	"jiqiren/bot/database/model"
)

func CreateAuditLog(entry *model.AuditLog) error {
	db := database.GetDb()
	return db.Create(entry).Error
}

// FindAuditLogsByFractionSince returns the fraction's audit trail newer
// than `since`, newest first. The table itself is never pruned — callers
// decide the visible window (the panel shows the last two weeks).
func FindAuditLogsByFractionSince(fractionId uint, since time.Time) ([]model.AuditLog, error) {
	db := database.GetDb()
	logs := []model.AuditLog{}
	err := db.Where("fraction_id = ? AND created_at >= ?", fractionId, since).
		Order("created_at DESC").Find(&logs).Error
	if (err != nil) {
		return nil, err
	}

	return logs, nil
}
