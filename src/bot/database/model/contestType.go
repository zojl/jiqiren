package model

type ContestType struct {
	Id uint `gorm:"primaryKey"`
	Name string
	// The size tag here was malformed (no closing quote), so gorm never
	// applied it and the column is plain text; dropped rather than fixed to
	// avoid an unplanned column type change on existing databases.
	Code string
}