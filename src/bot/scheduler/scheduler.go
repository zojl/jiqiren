package scheduler

import (
	"log"
	"time"

	"jiqiren/bot/battletime"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/api/params"
)

var moscow = battletime.Moscow

var vk *api.VK

func SetVK(client *api.VK) {
	vk = client
}

func Start() {
	go run()
}

// run ticks once per wall-clock minute, right at its :00 mark — a plain
// time.Ticker instead would drift by up to its own interval, since it
// starts counting from whenever the process happened to launch rather
// than from the minute boundary.
func run() {
	for {
		tick()
		time.Sleep(untilNextMinute())
	}
}

func untilNextMinute() time.Duration {
	now := time.Now()
	next := now.Truncate(time.Minute).Add(time.Minute)
	return next.Sub(now)
}

func tick() {
	messages, err := repository.FindAllScheduledMessages()
	if (err != nil) {
		log.Println("scheduler: " + err.Error())
		return
	}

	now := time.Now()
	nowMsk := now.In(moscow)
	today := battletime.Today(now)

	for _, message := range messages {
		if (!isDue(message, nowMsk, today)) {
			continue
		}

		squad, squadErr := repository.FindOneTeamById(message.SquadId)
		if (squadErr != nil || squad.ChatId == nil) {
			continue // nothing to send to yet; try again next tick
		}

		if sendErr := dispatch(message, squad, now); (sendErr != nil) {
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

// dispatch builds the final text and sends it. For generated types, the
// configured Text is an optional prefix — generated content is appended
// after it. If there are no mentions, the prefix alone still goes out as
// long as it's non-empty; only a fully empty result (no prefix, nothing
// generated) is skipped, though the slot still counts as handled today.
func dispatch(message model.ScheduledMessage, squad *model.Team, now time.Time) error {
	if (message.Type == model.ScheduledMessageTypeBasic) {
		return sendText(*squad.ChatId, message.Text)
	}

	var generated string
	switch (message.Type) {
	case model.ScheduledMessageTypeMissingProfiles:
		generated = generateMissingProfilesText(squad, now)
	case model.ScheduledMessageTypeMissingReports:
		generated = generateMissingReportsText(squad, now)
	case model.ScheduledMessageTypeSleepReminder:
		generated = generateSleepReminderText(squad, now)
	default:
		return nil
	}

	combined := combineText(message.Text, generated)
	if (combined == "") {
		return nil
	}

	return sendText(*squad.ChatId, combined)
}

func combineText(prefix string, generated string) string {
	if (prefix == "") {
		return generated
	}

	if (generated == "") {
		return prefix
	}

	return prefix + "\n" + generated
}

func sendText(peerId int64, text string) error {
	builder := params.NewMessagesSendBuilder()
	builder.Message(text)
	builder.RandomID(0)
	builder.PeerID(int(peerId))

	_, err := vk.MessagesSend(builder.Params)
	return err
}
