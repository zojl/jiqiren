package model

import "time"

type AccountType string

const (
	AccountTypeTeamlead  AccountType = "teamlead"
	AccountTypeDeputy    AccountType = "deputy"
	AccountTypeSquadLead AccountType = "squad_lead"
)

type Manager struct {
	Id uint `gorm:"primaryKey"`
	VkId uint `gorm:"uniqueIndex"`
	FractionId uint
	Fraction Fraction `gorm:"foreignKey:FractionId"`
	SquadId *uint `gorm:"null"`
	Squad Team `gorm:"foreignKey:SquadId"`
	AccountType *AccountType `gorm:"null;size:16"`
	LoginCode *string `gorm:"null;size:6"`
	LoginCodeExpiresAt *time.Time `gorm:"null"`
}
