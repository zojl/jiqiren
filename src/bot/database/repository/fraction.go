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

func SetFractionMessage(fraction *model.Fraction, message string) error {
	db := database.GetDb()
	return db.Model(fraction).Update("Message", message).Error
}

// SetFractionReadinessSettings writes all six thresholds in one go; a nil
// pointer clears that column back to NULL (readiness.ThresholdsFor falls
// back to its default), same "empty clears it" convention as the message
// form's delete behavior.
func SetFractionReadinessSettings(fraction *model.Fraction, staminaVerySevere *uint, staminaSevere *uint, staminaRecommended *uint, moneyRecommended *uint, moneySevere *uint, moneyVerySevere *uint) error {
	db := database.GetDb()

	if err := db.Model(fraction).Update("StaminaVerySevereThreshold", staminaVerySevere).Error; (err != nil) {
		return err
	}
	if err := db.Model(fraction).Update("StaminaSevereThreshold", staminaSevere).Error; (err != nil) {
		return err
	}
	if err := db.Model(fraction).Update("StaminaRecommendedThreshold", staminaRecommended).Error; (err != nil) {
		return err
	}
	if err := db.Model(fraction).Update("MoneyRecommendedThreshold", moneyRecommended).Error; (err != nil) {
		return err
	}
	if err := db.Model(fraction).Update("MoneySevereThreshold", moneySevere).Error; (err != nil) {
		return err
	}

	return db.Model(fraction).Update("MoneyVerySevereThreshold", moneyVerySevere).Error
}