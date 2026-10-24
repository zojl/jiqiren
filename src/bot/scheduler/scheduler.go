package scheduler

import (
	"log"
	"time"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/api/params"
)

// moscow is a fixed +3 offset rather than a tzdata-backed location, same
// rationale as panel/handlers/squad.go: Moscow hasn't observed DST since
// 2014, and the prod image isn't guaranteed to ship tzdata.
var moscow = time.FixedZone("MSK", 3*60*60)

const tickInterval = 30 * time.Second

var vk *api.VK

func SetVK(client *api.VK) {
	vk = client
}

func Start() {
	go run()
}

func run() {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		tick()
		<-ticker.C
	}
}

func tick() {
	messages, err := repository.FindAllScheduledMessages()
	if (err != nil) {
		log.Println("scheduler: " + err.Error())
		return
	}

	nowMsk := time.Now().In(moscow)
	today := time.Date(nowMsk.Year(), nowMsk.Month(), nowMsk.Day(), 0, 0, 0, 0, time.UTC)

	for _, message := range messages {
		if (!isDue(message, nowMsk, today)) {
			continue
		}

		squad, squadErr := repository.FindOneTeamById(message.SquadId)
		if (squadErr != nil || squad.ChatId == nil) {
			continue // nothing to send to yet; try again next tick
		}

		if sendErr := dispatch(message, squad); (sendErr != nil) {
			log.Println("scheduler: " + sendErr.Error())
			continue
		}

		if markErr := repository.MarkScheduledMessageSent(&message, today); (markErr != nil) {
			log.Println("scheduler: " + markErr.Error())
		}
	}
}

func isDue(message model.ScheduledMessage, nowMsk time.Time, today time.Time) bool {
	if (message.LastSentDate != nil && message.LastSentDate.Equal(today)) {
		return false
	}

	sendMoment := time.Date(nowMsk.Year(), nowMsk.Month(), nowMsk.Day(), int(message.SendHour), int(message.SendMinute), 0, 0, moscow)

	return !nowMsk.Before(sendMoment)
}

func dispatch(message model.ScheduledMessage, squad *model.Team) error {
	switch (message.Type) {
	case model.ScheduledMessageTypeBasic:
		return sendText(*squad.ChatId, message.Text)
	default:
		return nil
	}
}

func sendText(peerId int64, text string) error {
	builder := params.NewMessagesSendBuilder()
	builder.Message(text)
	builder.RandomID(0)
	builder.PeerID(int(peerId))

	_, err := vk.MessagesSend(builder.Params)
	return err
}
