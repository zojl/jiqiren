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
	// default", see readiness.ThresholdsFor. Stamina's "ideal" cutoff isn't
	// here — it's fixed at readiness.StaminaIdealThreshold, not overridable.
	StaminaVerySevereThreshold  *uint `gorm:"null"`
	StaminaSevereThreshold      *uint `gorm:"null"`
	StaminaRecommendedThreshold *uint `gorm:"null"`
	MoneyRecommendedThreshold   *uint `gorm:"null"`
	MoneySevereThreshold        *uint `gorm:"null"`
	MoneyVerySevereThreshold    *uint `gorm:"null"`
	// Gear thresholds are in absolute ⚔️/🛡️ points the player falls short
	// of their optimal item set (see the gearcheck package).
	GearWarnThreshold       *uint `gorm:"null"`
	GearSevereThreshold     *uint `gorm:"null"`
	GearVerySevereThreshold *uint `gorm:"null"`
}