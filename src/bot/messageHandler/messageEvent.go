package messageHandler

import (
	"encoding/json"
	"log"

	"jiqiren/bot/messageHandler/command"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/api/params"
	"github.com/SevereCloud/vksdk/v2/events"
	"github.com/SevereCloud/vksdk/v2/object"
)

type messageEventPayload struct {
	Cmd string `json:"cmd"`
}

// HandleMessageEvent answers a tap on a callback-type keyboard button (see
// mainKeyboard in keyboard.go) and runs the command it stands for —
// the button's visible label is purely cosmetic, routing goes through the
// payload instead, so it can't be spoofed by someone just typing the label.
func HandleMessageEvent(eventObject events.MessageEventObject, vk *api.VK) {
	answer := params.NewMessagesSendMessageEventAnswerBuilder()
	answer.EventID(eventObject.EventID)
	answer.UserID(eventObject.UserID)
	answer.PeerID(eventObject.PeerID)
	if _, err := vk.MessagesSendMessageEventAnswer(answer.Params); (err != nil) {
		log.Println("failed to answer message event: " + err.Error())
	}

	var payload messageEventPayload
	if err := json.Unmarshal(eventObject.Payload, &payload); (err != nil) {
		return
	}

	syntheticMessage := &events.MessageNewObject{
		Message: object.MessagesMessage{
			FromID: eventObject.UserID,
			PeerID: eventObject.PeerID,
			ConversationMessageID: eventObject.ConversationMessageID,
		},
	}

	if (isInventoryCommand(payload.Cmd)) {
		handleInventoryCommand(payload.Cmd, &syntheticMessage.Message, vk)
		return
	}

	switch (payload.Cmd) {
	case "profile":
		command.Profile{Message: syntheticMessage, Vk: vk}.Handle()
	case "login":
		command.Login{Message: syntheticMessage, Vk: vk}.Handle()
	}
}
