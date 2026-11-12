package model

import "time"

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

	// Panel identity — anyone with a profile (or a manager row) can log
	// in, so these live here rather than on Manager. SessionId must match
	// the session cookie's sid (see auth.RequireAuth); rotating it ends
	// every session at once.
	LoginCode *string `gorm:"null;size:8"`
	LoginCodeExpiresAt *time.Time `gorm:"null"`
	SessionId *string `gorm:"null;size:64"`
	ThemeMode *string `gorm:"null;size:8"`
	ColorScheme *string `gorm:"null;size:16"`
}
