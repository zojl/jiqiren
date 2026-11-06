package model

type Fraction struct {
	Id uint `gorm:"primaryKey"`
	Name string
	Code string `gorm:"size:2`
	Icon string
	TargetToken string `gorm:"size:255"`
	Target *uint `gorm:"null"`
	Message string `gorm:"type:text"`

	// Readiness thresholds — nil means "use the readiness package's
	// default", see readiness.ThresholdsFor.
	StaminaSevereThreshold      *uint `gorm:"null"`
	StaminaRecommendedThreshold *uint `gorm:"null"`
	MoneyRecommendedThreshold   *uint `gorm:"null"`
	MoneySevereThreshold        *uint `gorm:"null"`
}