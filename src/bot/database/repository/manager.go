package repository

import (
	"fmt"
	"time"

	"jiqiren/bot/database"
	"jiqiren/bot/database/model"
)

func FindOneManagerByVkId(vkId uint) (*model.Manager, error) {
	db := database.GetDb()
	manager := model.Manager{}
	err := db.Unscoped().Where("vk_id = ?", vkId).First(&manager).Error
	if (err != nil) {
		return nil, fmt.Errorf("No manager associated with vk id: %d", vkId)
	}

	return &manager, nil
}

func FindOneManagerById(id uint) (*model.Manager, error) {
	db := database.GetDb()
	manager := model.Manager{}
	err := db.Unscoped().Where("id = ?", id).First(&manager).Error
	if (err != nil) {
		return nil, fmt.Errorf("No manager with id: %d", id)
	}

	return &manager, nil
}

func FindOneManagerByLoginCode(code string) (*model.Manager, error) {
	db := database.GetDb()
	manager := model.Manager{}
	err := db.Unscoped().Where("login_code = ? AND login_code_expires_at > ?", code, time.Now()).First(&manager).Error
	if (err != nil) {
		return nil, fmt.Errorf("Invalid or expired login code")
	}

	return &manager, nil
}

func FindManagersByFractionId(fractionId uint) ([]model.Manager, error) {
	db := database.GetDb()
	managers := []model.Manager{}
	err := db.Unscoped().Where("fraction_id = ?", fractionId).Find(&managers).Error
	if (err != nil) {
		return nil, fmt.Errorf("Could not load managers for fraction id: %d", fractionId)
	}

	return managers, nil
}

func CreateManager(manager *model.Manager) error {
	db := database.GetDb()
	return db.Create(manager).Error
}

func UpdateManagerRole(manager *model.Manager) error {
	db := database.GetDb()
	return db.Model(manager).Select("AccountType", "SquadId").Updates(model.Manager{
		AccountType: manager.AccountType,
		SquadId: manager.SquadId,
	}).Error
}

func DeleteManager(id uint) error {
	db := database.GetDb()
	return db.Delete(&model.Manager{}, id).Error
}

func SetManagerLoginCode(manager *model.Manager, code string, expiresAt time.Time) error {
	db := database.GetDb()
	return db.Model(manager).Updates(map[string]interface{}{
		"login_code": code,
		"login_code_expires_at": expiresAt,
	}).Error
}

func ClearManagerLoginCode(manager *model.Manager) error {
	db := database.GetDb()
	return db.Model(manager).Updates(map[string]interface{}{
		"login_code": nil,
		"login_code_expires_at": nil,
	}).Error
}
