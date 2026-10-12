package repository

import (
	"fmt"

	"jiqiren/bot/database"
	"jiqiren/bot/database/model"
)

func FindOneTeamById(id uint) (*model.Team, error) {
	db := database.GetDb()
	team := model.Team{}
	err := db.Unscoped().Where("id = ?", id).First(&team).Error
	if (err != nil) {
		return nil, fmt.Errorf("No squad with id: %d", id)
	}

	return &team, nil
}

func FindTeamsByFractionId(fractionId uint) ([]model.Team, error) {
	db := database.GetDb()
	teams := []model.Team{}
	err := db.Unscoped().Where("fraction_id = ?", fractionId).
		Order("updated_at DESC NULLS LAST, id ASC").Find(&teams).Error
	if (err != nil) {
		return nil, fmt.Errorf("Could not load squads for fraction id: %d", fractionId)
	}

	return teams, nil
}

func CreateTeam(team *model.Team) error {
	db := database.GetDb()
	return db.Create(team).Error
}

func RenameTeam(team *model.Team, name string) error {
	db := database.GetDb()
	return db.Model(team).Update("Name", name).Error
}

func DeleteTeam(id uint) error {
	db := database.GetDb()
	return db.Delete(&model.Team{}, id).Error
}

func SetTeamToken(team *model.Team, token string) error {
	db := database.GetDb()
	return db.Model(team).Update("TargetToken", token).Error
}

func SetTeamTarget(team *model.Team, target uint) error {
	db := database.GetDb()
	return db.Model(team).Update("Target", target).Error
}
