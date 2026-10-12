package model

type Fraction struct {
	Id uint `gorm:"primaryKey"`
	Name string
	Code string `gorm:"size:2`
	Icon string
	TargetToken string `gorm:"size:255"`
	Target *uint `gorm:"null"`
}