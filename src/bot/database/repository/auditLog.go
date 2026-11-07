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

// AuditLogFilter narrows FindAuditLogs beyond fraction + time window — a
// nil field leaves that dimension unfiltered, and the two combine with AND
// when both are set.
type AuditLogFilter struct {
	ManagerId *uint
	Action    *model.AuditAction
}

func FindAuditLogs(fractionId uint, since time.Time, filter AuditLogFilter) ([]model.AuditLog, error) {
	db := database.GetDb()
	query := db.Where("fraction_id = ? AND created_at >= ?", fractionId, since)

	if (filter.ManagerId != nil) {
		query = query.Where("manager_id = ?", *filter.ManagerId)
	}

	if (filter.Action != nil) {
		query = query.Where("action = ?", *filter.Action)
	}

	logs := []model.AuditLog{}
	err := query.Order("created_at DESC").Find(&logs).Error
	if (err != nil) {
		return nil, err
	}

	return logs, nil
}
