package repository

import (
	"fmt"

	"jiqiren/bot/database"
	"jiqiren/bot/database/model"
)

func FindOneFractionByIcon(icon string) (*model.Fraction, error) {
	if (icon == "🔐") {
		return nil, nil
	}

	db := database.GetDb()
	fraction := model.Fraction{}
	err := db.Unscoped().Where("Icon = ?", icon).First(&fraction).Error
	if (err != nil) {
		return nil, fmt.Errorf("Erroneous fraction icon: %s", icon)
	}

	return &fraction, nil
}

func FindOneFractionById(id uint) (*model.Fraction, error) {
	db := database.GetDb()
	fraction := model.Fraction{}
	err := db.Unscoped().Where("id = ?", id).First(&fraction).Error
	if (err != nil) {
		return nil, fmt.Errorf("No fraction with id: %d", id)
	}

	return &fraction, nil
}

func FindAllFractions() ([]model.Fraction, error) {
	db := database.GetDb()
	fractions := []model.Fraction{}
	err := db.Unscoped().Order("id").Find(&fractions).Error
	if (err != nil) {
		return nil, fmt.Errorf("Could not load fractions")
	}

	return fractions, nil
}

func SetFractionToken(fraction *model.Fraction, token string) error {
	db := database.GetDb()
	return db.Model(fraction).Update("TargetToken", token).Error
}

func SetFractionTarget(fraction *model.Fraction, target uint) error {
	db := database.GetDb()
	return db.Model(fraction).Update("Target", target).Error
}