package repository

import (
	"fmt"

	"jiqiren/bot/database"
	"jiqiren/bot/database/model"
)

func FindOneContestTypeById(id uint) (*model.ContestType, error) {
	db := database.GetDb()
	contestType := model.ContestType{}
	err := db.Unscoped().Where("Id = ?", id).First(&contestType).Error
	if (err != nil) {
		return nil, fmt.Errorf("Erroneous contest id: %d", id)
	}

	return &contestType, nil
}