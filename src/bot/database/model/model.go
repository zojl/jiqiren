package model

import (
	"log"

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
		BudgetSnapshot{},
		PlannedTarget{},
		InventoryDraft{},
		InventoryDraftMessage{},
		InventoryItem{},
		InventoryAnalysis{},
	}
}

func CreateInitialValues(db *gorm.DB) {
	migrateManagerIdentityToUsers(db)
	dropObsoleteIndexes(db)
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
// migrateManagerIdentityToUsers is a one-off move of the panel identity
// columns from managers to users (panel login is no longer manager-only).
// Only theme preferences are carried over — login codes are short-lived
// anyway, and session ids are deliberately dropped: old cookies carried a
// manager id, not a user id, so everyone simply logs in again once.
// Idempotent: once the old columns are gone, it's a no-op.
func migrateManagerIdentityToUsers(db *gorm.DB) {
	migrator := db.Migrator()
	if (!migrator.HasColumn(&Manager{}, "theme_mode")) {
		return
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`INSERT INTO users (vk_id, frozen)
			SELECT m.vk_id, false FROM managers m
			ON CONFLICT (vk_id) DO NOTHING`).Error; (err != nil) {
			return err
		}

		if err := tx.Exec(`UPDATE users u
			SET theme_mode = m.theme_mode, color_scheme = m.color_scheme
			FROM managers m
			WHERE m.vk_id = u.vk_id AND u.theme_mode IS NULL AND m.theme_mode IS NOT NULL`).Error; (err != nil) {
			return err
		}

		for _, column := range []string{"login_code", "login_code_expires_at", "theme_mode", "color_scheme", "session_id"} {
			if err := tx.Exec("ALTER TABLE managers DROP COLUMN IF EXISTS " + column).Error; (err != nil) {
				return err
			}
		}

		return nil
	})
	if (err != nil) {
		log.Printf("migrate manager identity to users: %v", err)
	}
}

// dropObsoleteIndexes removes indexes that a newer one has replaced —
// AutoMigrate creates missing indexes but never drops leftover ones.
func dropObsoleteIndexes(db *gorm.DB) {
	// Superseded by idx_profiles_team_user.
	if err := db.Exec("DROP INDEX IF EXISTS idx_profiles_team_id").Error; (err != nil) {
		log.Printf("drop obsolete index idx_profiles_team_id: %v", err)
	}
}
