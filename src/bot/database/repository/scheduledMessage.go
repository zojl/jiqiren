package repository

import (
	"fmt"
	"time"

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

func CreateScheduledMessage(message *model.ScheduledMessage) error {
	db := database.GetDb()
	return db.Create(message).Error
}

func UpdateScheduledMessage(message *model.ScheduledMessage, hour uint, minute uint, text string) error {
	db := database.GetDb()
	return db.Model(message).Updates(map[string]interface{}{
		"send_hour": hour,
		"send_minute": minute,
		"text": text,
	}).Error
}

func DeleteScheduledMessage(id uint) error {
	db := database.GetDb()
	return db.Delete(&model.ScheduledMessage{}, id).Error
}

func MarkScheduledMessageSent(message *model.ScheduledMessage, date time.Time) error {
	db := database.GetDb()
	return db.Model(message).Update("LastSentDate", date).Error
}
