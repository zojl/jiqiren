package messageHandler

import (
	"log"
	"os"
	"strings"

	"jiqiren/bot/messageHandler/command"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/events"
)

const ProfileButtonLabel = "🥷 Профиль"
const WebPanelButtonLabel = "🌐 Веб-панель"

// IsButtonCommand reports whether text is one of the keyboard button labels
// attached to the profile-ack message (see buildProfileAckKeyboard in
// plain.go) — those don't carry PREFIX, so the main dispatch loop needs this
// check alongside its usual prefix check to route a button tap into
// HandleCommand instead of HandlePlain.
func IsButtonCommand(text string) bool {
	return text == ProfileButtonLabel || text == WebPanelButtonLabel
}

func HandleCommand(messageObject *events.MessageNewObject, vk *api.VK) {
	command := getCommand(messageObject, vk)
	log.Printf("%+v\n", command)
	if (command != nil) {
		command.Handle()
	}
}

func getCommand(messageObject *events.MessageNewObject, vk *api.VK) Command {
	if isCommand(messageObject, []string{"contest", "конкурс"}) {
		return command.Contest{
			Message: messageObject,
			Vk: vk,
		}
	}

	if isCommand(messageObject, []string{"login"}) || messageObject.Message.Text == WebPanelButtonLabel {
		return command.Login{
			Message: messageObject,
			Vk: vk,
		}
	}

	if isCommand(messageObject, []string{"bind", "привязать"}) {
		return command.BindChat{
			Message: messageObject,
			Vk: vk,
		}
	}

	if isCommand(messageObject, []string{"profile", "профиль"}) || messageObject.Message.Text == ProfileButtonLabel {
		return command.Profile{
			Message: messageObject,
			Vk: vk,
		}
	}

	return nil
}

func isCommand(messageObject *events.MessageNewObject, commands []string) bool {
	prefix := os.Getenv("PREFIX")
	for _, command := range commands {
		log.Println(prefix + command)
		if strings.HasPrefix(messageObject.Message.Text, prefix + command + " ") {
			return true
		}
		
		if messageObject.Message.Text == prefix + command {
			return true
		}
	}
	
	return false
}

type Command interface {
	Handle()
}