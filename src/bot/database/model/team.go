package model

import "time"

type Team struct {
	Id uint `gorm:"primaryKey"`
	Name string
	// The size tag here was malformed (no closing quote), so gorm never
	// applied it and the column is plain text; dropped rather than fixed to
	// avoid an unplanned column type change on existing databases.
	Code string
	FractionId uint
	Fraction Fraction `gorm:"foreignKey:FractionId"`
	ChatId *int64 `gorm:"null"`
	ChatName *string `gorm:"null"`
	TargetToken string `gorm:"size:255"`
	Target *uint `gorm:"null"`
	UpdatedAt time.Time
}