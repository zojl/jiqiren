package model

type AccountType string

const (
	AccountTypeTeamlead  AccountType = "teamlead"
	AccountTypeDeputy    AccountType = "deputy"
	AccountTypeSquadLead AccountType = "squad_lead"
)

// Manager only grants a panel role on top of a User — the panel identity
// itself (login code, session id, theme) lives on User, so a manager row
// being deleted just demotes that person to a plain player's view.
type Manager struct {
	Id uint `gorm:"primaryKey"`
	VkId uint `gorm:"uniqueIndex"`
	FractionId uint
	Fraction Fraction `gorm:"foreignKey:FractionId"`
	SquadId *uint `gorm:"null"`
	Squad Team `gorm:"foreignKey:SquadId"`
	AccountType *AccountType `gorm:"null;size:16"`
}
