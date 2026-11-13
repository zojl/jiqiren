package model

import "time"

type Profile struct {
	Id uint `gorm:"primaryKey"`
	Username string
	Status string
	// (team_id, user_id) for squad membership — "who has ever been in
	// this squad" (repository.FindLatestProfilesForTeam) reads straight
	// from the index, without touching the table.
	TeamId *uint `gorm:"null;index:idx_profiles_team_user,priority:1,where:NOT hidden"`
	Team Team `gorm:"foreignKey:TeamId"`
	FractionId uint
	Fraction Fraction `gorm:"foreignKey:FractionId"`
	Lead uint
	Way uint
	Level uint
	Prestige uint
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
	// (user_id, message_date) serves nearly every profile read: a user's
	// latest profile, battle-date windows, history pages, the experience
	// chart and best stats all filter by user and most by date range.
	UserId uint `gorm:"index:idx_profiles_user_date,priority:1;index:idx_profiles_team_user,priority:2"`
	User User `gorm:"foreignKey:UserId"`
	MessageDate time.Time `gorm:"index:idx_profiles_user_date,priority:2"`
	SleepAt *time.Time `gorm:"null"`
	Created time.Time `gorm:"autoCreateTime"`
	// Hidden marks a profile its own sender disowned in "Мой профиль" —
	// typically someone else's profile forwarded by mistake. A hidden
	// profile is excluded from every read (latest profile, squad
	// membership, battle-date lookups, best stats, charts); it only
	// still shows in the sender's own history, to be restored.
	Hidden bool `gorm:"not null;default:false"`
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
