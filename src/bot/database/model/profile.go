package model

import "time"

type Profile struct {
	Id uint `gorm:"primaryKey"`
	Username string
	TeamId *uint `gorm:"null"`
	Team Team `gorm:"foreignKey:TeamId"`
	FractionId uint
	Fraction Fraction `gorm:"foreignKey:FractionId"`
	Lead uint
	Way uint
	Level uint
	Experience uint
	Money uint
	Vkcoin float64 `gorm:"type:decimal(21,6);"`
	Points uint
	Bitcoins uint
	Disks uint
	Pages uint
	Chips uint
	Instructions uint
	Stocks uint
	Motivation uint
	MotivationLimit uint
	Practice uint
	Theory uint
	Cunning uint
	Wisdom uint
	Stamina uint
	TargetId *uint `gorm:"null"`
	Target Fraction `gorm:"foreignKey:TargetId;null;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	UserId uint
	User User `gorm:"foreignKey:UserId"`
	MessageDate time.Time
	SleepAt *time.Time `gorm:"null"`
	Created time.Time `gorm:"autoCreateTime"`
}

// EffectiveSquadId is the squad this profile's user should count as
// belonging to for display/mentions — a manual override always wins over
// what the profile itself reports.
func (p Profile) EffectiveSquadId(overrideSquadId *uint) *uint {
	if (overrideSquadId != nil) {
		return overrideSquadId
	}

	return p.TeamId
}
