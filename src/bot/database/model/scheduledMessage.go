package model

import "time"

type ScheduledMessageType string

const (
	ScheduledMessageTypeBasic ScheduledMessageType = "basic"
)

// LastSentDate guards against sending twice on the same MSK calendar day —
// the scheduler ticks far more often than once a day and must tolerate
// restarts without losing track of what already went out today.
type ScheduledMessage struct {
	Id uint `gorm:"primaryKey"`
	SquadId uint
	Squad Team `gorm:"foreignKey:SquadId"`
	Type ScheduledMessageType `gorm:"size:32"`
	SendHour uint
	SendMinute uint
	Text string `gorm:"type:text"`
	LastSentDate *time.Time `gorm:"null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
