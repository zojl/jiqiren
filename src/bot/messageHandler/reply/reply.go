package reply

import (
	"encoding/json"
	"log"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/object"
	"github.com/SevereCloud/vksdk/v2/api/params"
)

func MakeParams(sourceMessage *object.MessagesMessage, vk *api.VK) *Params {
	params := Params{
		sourceMessage: sourceMessage,
		vk: vk,
	}

	return &params
}

type Params struct{
	sourceMessage *object.MessagesMessage
	vk *api.VK
}

func (replyParams Params) Reply(text string, ) {
	replyParams.ReplyWithKeyboard(text, nil)
}

func (replyParams Params) ReplyWithKeyboard(text string, keyboard *object.MessagesKeyboard) {
	messageBuilder := params.NewMessagesSendBuilder()
	messageBuilder.Message(text)
	messageBuilder.RandomID(0)
	messageBuilder.PeerID(replyParams.sourceMessage.PeerID)

	// A button tap on a non-inline keyboard carries no originating message
	// (ConversationMessageID is 0 — there's nothing to quote-reply to), so
	// only attach the reply-forward when sourceMessage actually points at one.
	if (replyParams.sourceMessage.ConversationMessageID > 0) {
		forward := Forward{}
		forward.IsReply = 1
		forward.PeerID = replyParams.sourceMessage.PeerID
		messageIds := []int{replyParams.sourceMessage.ConversationMessageID}
		forward.ConversationMessageIDs = messageIds
		jsonMessageRaw, _ := json.Marshal(forward)
		jsonMessage := string(jsonMessageRaw)
		messageBuilder.Forward(jsonMessage)
	}

	if (keyboard != nil) {
		messageBuilder.Keyboard(keyboard.ToJSON())
	}

	_, err := replyParams.vk.MessagesSend(messageBuilder.Params)
	if err != nil {
		log.Fatal(err)
	}
}

type Forward struct {
	IsReply int `json:"is_reply"`
	PeerID int `json:"peer_id"`
	ConversationMessageIDs []int `json:"conversation_message_ids"`
}
