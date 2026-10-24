package repository

import (
	"fmt"
	"time"

	"jiqiren/bot/battletime"
	"jiqiren/bot/database"
	"jiqiren/bot/database/model"
)

func FindAllScheduledMessages() ([]model.ScheduledMessage, error) {
	db := database.GetDb()
	messages := []model.ScheduledMessage{}
	err := db.Find(&messages).Error
	if (err != nil) {
		return nil, fmt.Errorf("Could not load scheduled messages")
	}

	return messages, nil
}

func FindScheduledMessagesBySquadId(squadId uint) ([]model.ScheduledMessage, error) {
	db := database.GetDb()
	messages := []model.ScheduledMessage{}
	err := db.Where("squad_id = ?", squadId).Order("send_hour, send_minute").Find(&messages).Error
	if (err != nil) {
		return nil, fmt.Errorf("Could not load scheduled messages for squad id: %d", squadId)
	}

	return messages, nil
}

func FindOneScheduledMessageById(id uint) (*model.ScheduledMessage, error) {
	db := database.GetDb()
	message := model.ScheduledMessage{}
	err := db.Where("id = ?", id).First(&message).Error
	if (err != nil) {
		return nil, fmt.Errorf("No scheduled message with id: %d", id)
	}

	return &message, nil
}

// CreateScheduledMessage pre-marks today as already sent when the chosen
// time has already gone by today — otherwise the scheduler would read a
// past, never-sent time as overdue and fire immediately instead of
// waiting for tomorrow.
func CreateScheduledMessage(message *model.ScheduledMessage) error {
	now := time.Now()
	if (battletime.IsPastToday(message.SendHour, message.SendMinute, now)) {
		today := battletime.Today(now)
		message.LastSentDate = &today
	}

	db := database.GetDb()
	return db.Create(message).Error
}

// UpdateScheduledMessage re-derives LastSentDate whenever the send time
// actually changes: cleared if the new time is still ahead today (so a
// message already sent under the old time can fire again today at the
// corrected time), or set to today if the new time has already gone by
// (so it doesn't read as overdue and fire immediately — it waits for
// tomorrow, same as a freshly created message with a past time).
func UpdateScheduledMessage(message *model.ScheduledMessage, messageType model.ScheduledMessageType, hour uint, minute uint, text string) error {
	db := database.GetDb()
	updates := map[string]interface{}{
		"type": messageType,
		"send_hour": hour,
		"send_minute": minute,
		"text": text,
	}

	if (hour != message.SendHour || minute != message.SendMinute) {
		now := time.Now()
		if (battletime.IsPastToday(hour, minute, now)) {
			updates["last_sent_date"] = battletime.Today(now)
		} else {
			updates["last_sent_date"] = nil
		}
	}

	return db.Model(message).Updates(updates).Error
}

func DeleteScheduledMessage(id uint) error {
	db := database.GetDb()
	return db.Delete(&model.ScheduledMessage{}, id).Error
}

func MarkScheduledMessageSent(message *model.ScheduledMessage, date time.Time) error {
	db := database.GetDb()
	return db.Model(message).Update("LastSentDate", date).Error
}
