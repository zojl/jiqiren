package model

type User struct {
	Id uint `gorm:"primaryKey"`
	VkId uint `gorm:"unique"`
	// SquadOverrideId, when set, overrides which squad this user counts
	// as belonging to everywhere (member lists, mentions) — regardless
	// of what their own profile reports. Reversible: nil restores the
	// profile-derived squad.
	SquadOverrideId *uint `gorm:"null"`
}