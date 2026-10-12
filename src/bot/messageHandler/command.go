package messageHandler

import (
	"log"
	"os"
	"strings"

	"jiqiren/bot/messageHandler/command"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/events"
)

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

	if isCommand(messageObject, []string{"login"}) {
		return command.Login{
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