package model

import "time"

// InventoryDraft is an inventory analysis in progress in the bot (one per
// user at most). It only collects raw forwarded messages — the user's
// actual inventory (InventoryItem) is untouched until they confirm.
type InventoryDraft struct {
	Id uint `gorm:"primaryKey"`
	UserId uint `gorm:"uniqueIndex"`
	// StartedAt — forwards of game messages sent before this are rejected,
	// so a stale screen can't sneak into a fresh analysis.
	StartedAt time.Time
	// Confirming is set while the "провести / продолжить / отменить"
	// confirmation is on screen; it only decides which keyboard is shown.
	Confirming bool `gorm:"not null;default:false"`
}

// InventoryDraftMessage is one collected game screen, raw — parsed on
// demand (see the inventory package), so a parser fix applies to drafts
// already in progress. Re-forwarding the same screen page replaces it.
type InventoryDraftMessage struct {
	Id uint `gorm:"primaryKey"`
	DraftId uint `gorm:"uniqueIndex:idx_inventory_draft_message_page,priority:1"`
	Kind string `gorm:"size:16;uniqueIndex:idx_inventory_draft_message_page,priority:2"`
	Page uint `gorm:"uniqueIndex:idx_inventory_draft_message_page,priority:3"`
	Text string `gorm:"type:text"`
	// SentAt is when the game sent the message (the forward's own date).
	SentAt time.Time
}

// InventoryItem is one item of a user's analyzed inventory. A confirmed
// analysis replaces the user's whole set at once — items move between
// locations, so a partial update would leave stale ones behind.
type InventoryItem struct {
	Id uint `gorm:"primaryKey"`
	UserId uint `gorm:"index"`
	Location string `gorm:"size:16"`

	Rarity string `gorm:"size:8"`
	IsServer bool
	Class string `gorm:"size:8"`
	Name string
	Way string `gorm:"size:8"`
	Practice int
	Theory int
	Cunning int
	Wisdom int

	FirmwareRarity string `gorm:"size:8"`
	FirmwareName string
	RequiredLevel uint
	Price uint
	Unsellable bool
	// Counters are " - Побед в битвах: 0" style lines, newline-joined.
	Counters string `gorm:"type:text"`
	Description string `gorm:"type:text"`
	Raw string `gorm:"type:text"`
}

// InventoryAnalysis records when a user's inventory was last analyzed.
// HasBackpack is false when that analysis had no equipped-items screen,
// so "what's worn now" (and gains over it) is unknown.
type InventoryAnalysis struct {
	Id uint `gorm:"primaryKey"`
	UserId uint `gorm:"uniqueIndex"`
	AnalyzedAt time.Time
	HasBackpack bool
}
