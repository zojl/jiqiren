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

// PrivateKeyboard, when set, supplies the keyboard for every reply in a
// private chat that doesn't pass its own — so the menu buttons refresh on
// any interaction with the bot (and new buttons show up after an update),
// not only when a profile is accepted. Set by messageHandler at init, so
// this package stays free of database imports. Group chats never get it:
// a keyboard there would show for every member.
var PrivateKeyboard func(vkId int) *object.MessagesKeyboard

// vkChatPeerOffset — peer ids at or above it are group chats.
const vkChatPeerOffset = 2000000000

func (replyParams Params) isPrivate() bool {
	peerId := replyParams.sourceMessage.PeerID
	return peerId > 0 && peerId < vkChatPeerOffset && peerId == replyParams.sourceMessage.FromID
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

	if (keyboard == nil && PrivateKeyboard != nil && replyParams.isPrivate()) {
		keyboard = PrivateKeyboard(replyParams.sourceMessage.FromID)
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
