package model

import "time"

type Report struct {
	Id uint `gorm:"primaryKey"`
	TargetId *uint `gorm:"null"`
	Target *Fraction `gorm:"foreignKey:TargetId;null;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	UserId uint
	User *User `gorm:"foreignKey:UserId"`
	BattleDate *time.Time
	IsSkipped bool
	IsSuccess bool
	IsAttack bool
	RewardMoney uint
	RewardExperience uint
	RewardVkc float64 `gorm:"type:decimal(16,6);"`
	Stamina uint
	TransactionMoney uint
	TransactionExperience uint
	LostMoney uint
	AttackLostMoney uint
	MessageDate time.Time
	Created time.Time `gorm:"autoCreateTime"`
}