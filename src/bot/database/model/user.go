package model

type User struct {
	Id uint `gorm:"primaryKey"`
	VkId uint `gorm:"unique"`
	// SquadOverrideId, when set, overrides which squad this user counts
	// as belonging to everywhere (member lists, mentions) — regardless
	// of what their own profile reports. Reversible: nil restores the
	// profile-derived squad.
	SquadOverrideId *uint `gorm:"null"`
	// Frozen, when manually set true, excludes this user from the bot's
	// missing-profile/report/sleep reminder pings even though they're not
	// yet naturally stale — e.g. they announced an absence in advance.
	// Cleared automatically the moment a fresh profile or report arrives.
	Frozen bool
}