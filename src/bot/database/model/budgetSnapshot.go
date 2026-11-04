package model

import "time"

type BudgetSnapshotKind string

const (
	BudgetSnapshotKindInterval   BudgetSnapshotKind = "interval"
	BudgetSnapshotKindPreBattle  BudgetSnapshotKind = "pre_battle"
	BudgetSnapshotKindPostBattle BudgetSnapshotKind = "post_battle"
)

// CapturedAt is the nominal MSK slot time this snapshot represents, not the
// exact HTTP response time — the scheduler tick already fires right on the
// minute boundary, so slot and fact coincide, and lookups by Kind + date
// don't need an approximate nearest-match search. The unique index guards
// against double-writing the same slot if the scheduler process restarts
// mid-minute.
type BudgetSnapshot struct {
	Id                 uint `gorm:"primaryKey"`
	FractionId         uint `gorm:"uniqueIndex:idx_budget_snapshot_fraction_time,priority:1"`
	Fraction           Fraction `gorm:"foreignKey:FractionId"`
	Kind               BudgetSnapshotKind `gorm:"size:16"`
	CapturedAt         time.Time `gorm:"uniqueIndex:idx_budget_snapshot_fraction_time,priority:2"`
	Budget             uint64
	ContributionsToday uint64
	ContributionsTotal uint64
	IncomeToday        uint64
	IncomeTotal        uint64
	CreatedAt          time.Time `gorm:"autoCreateTime"`
}
