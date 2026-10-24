package repository

import (
	"time"

	"jiqiren/bot/database"
	"jiqiren/bot/database/model"
)

func FindReportsByUserIdsAndDate(userIds []uint, date *time.Time) ([]model.Report, error) {
	db := database.GetDb()
	reports := []model.Report{}
	err := db.Where("user_id IN ? AND battle_date = ?", userIds, date).Find(&reports).Error
	if (err != nil) {
		return nil, err
	}

	return reports, nil
}

func CountReportsByUserAndDate(vkId int, date *time.Time) int64 {
	db := database.GetDb()
	var count int64
	err := db.Model(&model.Report{}).Joins("JOIN users ON users.id = reports.user_id").
		Where("users.vk_id = ? AND reports.battle_date = ?", vkId, date).Count(&count).Error

	if (err != nil) {
		return 0
	}

	return count
}