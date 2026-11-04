package model

import (
	"gorm.io/gorm"
)

func GetAllModels() []interface{} {
	return []interface{} {
		Fraction{},
		User{},
		Team{},
		Manager{},
		Profile{},
		Report{},
		ContestType{},
		Contest{},
		ContestPoints{},
		ContestProjectMessages{},
		AuditLog{},
		ScheduledMessage{},
	}
}

func CreateInitialValues(db *gorm.DB) {
	fillFractions(db)
	fillContestTypes(db)
}

func fillFractions(db *gorm.DB) {
	var existingFractionsCount int64
	db.Model(&Fraction{}).Count(&existingFractionsCount)
	if (existingFractionsCount >= 6) {
		return
	}

	fractions := [...]Fraction {
		{Name: "Aegis", Code: "ae", Icon: "💠"},
		{Name: "V-hack", Code: "vh", Icon: "🚧"},
		{Name: "Phantoms", Code: "ph", Icon: "🎭"},
		{Name: "Huǒqiáng", Code: "hu", Icon: "🈵"},
		{Name: "NetKings", Code: "nk", Icon: "🔱"},
		{Name: "NHS", Code: "nh", Icon: "🇺🇸"},
	}

	tableName := db.Model(&Fraction{}).Name()
	db.Exec("TRUNCATE TABLE " + tableName + " RESTART IDENTITY;")

	db.Create(&fractions)
}

func fillContestTypes(db *gorm.DB) {
	contestTypes := [...]ContestType {
		{Name: "Activity Contest", Code: "activity"},
		{Name: "Project Contest", Code: "project"},
	}

	for _, contestType := range contestTypes {
		var existingType = ContestType{}
		err := db.Unscoped().Model(&ContestType{}).Where("code = ?", contestType.Code).First(&existingType).Error
		if (err != nil || existingType.Id == 0) {
			db.Create(&contestType)
		}
	}
}