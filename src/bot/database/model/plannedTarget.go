package model

import "time"

type PlannedTargetType string

const (
	PlannedTargetTypeOnce      PlannedTargetType = "once"
	PlannedTargetTypeRecurring PlannedTargetType = "recurring"
)

// PlannedTarget is a planned future change of a squad's or fraction's
// Target — a scheduled counterpart to the manual "set target" admin
// action. SquadId nil means a fraction-scope entry; FractionId is always
// set (even for a squad entry, duplicating Squad.FractionId) so permission
// checks and fraction-wide listings don't need a join.
//
// LastAppliedDate guards against double-applying on the same MSK calendar
// day, same role as ScheduledMessage.LastSentDate — for a "once" entry it
// simply marks "already applied" regardless of which day that was.
// LastError is cleared by the next successful application.
type PlannedTarget struct {
	Id              uint `gorm:"primaryKey"`
	FractionId      uint
	Fraction        Fraction `gorm:"foreignKey:FractionId"`
	SquadId         *uint    `gorm:"null"`
	Squad           Team     `gorm:"foreignKey:SquadId"`
	Target          uint
	Type            PlannedTargetType `gorm:"size:16"`
	StartDate       time.Time
	EndDate         *time.Time `gorm:"null"`
	Hour            uint
	Minute          uint
	Reserve         bool
	Message         string     `gorm:"type:text"`
	LastAppliedDate *time.Time `gorm:"null"`
	LastError       string     `gorm:"type:text"`
	CreatedAt       time.Time  `gorm:"autoCreateTime"`
}
