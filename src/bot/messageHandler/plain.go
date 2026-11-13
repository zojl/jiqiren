package messageHandler

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"jiqiren/bot/inventory"
	"jiqiren/bot/messageHandler/plain"
	"jiqiren/bot/messageHandler/reply"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/object"
	"github.com/SevereCloud/vksdk/v2/events"
)

const battleReportStart = "Результаты битвы за "
const battleReportEnd = "защиту своей компании."
const teamProjectResultPrefix1 = "Ты вложился в запил командного проекта."
const teamProjectResultPrefix2 = "Ты помог команде пилить проект."
// profileIcons are the ones every profile has, whatever the player's
// level: resource lines (💿 📄 💽 📑) only appear once a player has some,
// so a low-level profile without them must still count as a profile.
var profileIcons = [...]string {"💻", "💡", "💵", "🔋", "📡", "💾", "📱", "🔎"}

func HandlePlain(messageObject events.MessageNewObject, vk *api.VK) {
	// Inventory screens are set aside and handled together, so forwarding
	// several pages in one message gets a single reply rather than one per
	// page.
	inventoryForwards := []object.MessagesMessage{}
	unrecognized := 0
	recognized := 0

	for _, message := range messageObject.Message.FwdMessages {
		if (strconv.Itoa(message.FromID) != os.Getenv("HW_ID")) {
			continue
		}

		if _, isInventory := inventory.DetectKind(message.Text); (isInventory) {
			inventoryForwards = append(inventoryForwards, message)
			continue
		}

		if (!isProfileMessage(message.Text) && !isBattleReport(message.Text) && !isTeamProjectResult(message.Text)) {
			unrecognized++
		} else {
			recognized++
		}

		handleHwForward(message, &messageObject.Message, vk)
	}

	// An unrecognized game message is normally ignored, but during an
	// inventory analysis silence would leave the player guessing whether
	// it was taken — so it's reported alongside the analysis progress.
	isPrivate := messageObject.Message.PeerID == messageObject.Message.FromID
	duringAnalysis := unrecognized > 0 && isPrivate && currentInventoryDraft(messageObject.Message.FromID) != nil

	if (len(inventoryForwards) > 0 || duringAnalysis) {
		handleInventoryForwards(inventoryForwards, unrecognized, &messageObject.Message, vk)
		return
	}

	if (recognized == 0) {
		replyUnrecognizedInPrivate(&messageObject.Message, vk)
	}
}

// replyUnrecognizedInPrivate answers a private message the bot couldn't
// make sense of — plain text (VK's "Начать" included), an unknown command,
// a forward that isn't from the game or isn't a known game screen. Beyond
// the hint itself, the reply carries the current menu keyboard (see
// reply.PrivateKeyboard), so writing anything at all refreshes the
// buttons. Group chats stay silent, as before.
func replyUnrecognizedInPrivate(message *object.MessagesMessage, vk *api.VK) {
	if (message.PeerID != message.FromID) {
		return
	}

	reply.MakeParams(message, vk).Reply(
		"Не понял сообщение. Пересылайте мне из игры профиль, отчёт о битве или — во время анализа — экраны инвентаря.\n" +
			"Кнопки ниже: сводка профиля, вход в веб-панель и анализ инвентаря.",
	)
}

func handleHwForward(message object.MessagesMessage, parentMessage *object.MessagesMessage, vk *api.VK) {
	messageDate := time.Unix(int64(message.Date), 0)
	senderId := parentMessage.FromID
	if (isProfileMessage(message.Text)) {
		if (os.Getenv("ENV") == "dev") {
			log.Println("That's a profile message from " + strconv.Itoa(senderId))
			log.Println("Contents: " + message.Text)
		}

		profileHandleResult := plain.HandleProfile(message.Text, senderId, messageDate)
		replyParams := reply.MakeParams(parentMessage, vk)

		if (profileHandleResult.IsInserted) {
			message := "Профиль принят."
			for _, messageNote := range profileHandleResult.Messages {
				message += "\n" + messageNote
			}

			isPrivate := parentMessage.PeerID == parentMessage.FromID
			if (isPrivate) {
				replyParams.ReplyWithKeyboard(message, mainKeyboard(senderId))
				return
			}

			replyParams.Reply(message)
			return
		}

		replyParams.Reply("Произошла ошибка, профиль не принят. Повторите ещё раз позже.")
		return
	}

	if (isBattleReport(message.Text)) {
		if (os.Getenv("ENV") == "dev") {
			log.Println("Battle report from " + strconv.Itoa(senderId))
			log.Println("Contents: " + message.Text)
		}

		reportResult := plain.HandleReport(message.Text, senderId, messageDate)

		replyParams := reply.MakeParams(parentMessage, vk)
		if (!reportResult.IsUserExist) {
			replyParams.Reply("Отчёт не принят, сначала нужно отправить боту профиль.")
			return
		}

		if (!reportResult.IsFirst) {
			replyParams.Reply("Отчёт не принят, повторный отчёт за эту дату.")
			return
		}

		if (!reportResult.IsStored) {
			replyParams.Reply("Отчёт не принят. Неизвестная ошибка.")
			return
		}

		if (!reportResult.IsParticipated) {
			replyParams.Reply("Отчёт принят. Участие в битвах важно для фракции, не пропускайте битвы!")
			return
		}

		if (!reportResult.IsProfileFound) {
			replyParams.Reply("Отчёт принят, но не найден профиль за период между этой битвой и прошлой. Не забывайте сдавать профиль.")
			return
		}

		message := "Отчёт принят. Спасибо за участие в битве!"
		if (reportResult.ContestResult != nil) {
			message = message + "\n" + reportResult.ContestResult.Message
		}
		replyParams.Reply(message)

		return
	}

	if (isTeamProjectResult(message.Text)) {
		if (os.Getenv("ENV") == "dev") {
			log.Println("team project result from " + strconv.Itoa(senderId))
			log.Println("Contents: " + message.Text)
		}

		projectResult := plain.HandleTeamProject(message, senderId, messageDate)

		if (projectResult) {
			replyParams := reply.MakeParams(parentMessage, vk)
			replyParams.Reply("Вклад в проект принят")
		}
	}

	log.Println("bad message " + message.Text)
}

func isProfileMessage(messageText string) bool {
	for _, icon := range profileIcons {
		if (!strings.Contains(messageText, icon)) {
			return false;
		}
	}

	return true
}

func isBattleReport(messageText string) bool {
	if !strings.HasPrefix(messageText, battleReportStart) {
		return false
	}

	return strings.HasSuffix(messageText, battleReportEnd)
}

func isTeamProjectResult(messageText string) bool {
	if strings.HasPrefix(messageText, teamProjectResultPrefix1) {
		return true
	}


	if strings.HasPrefix(messageText, teamProjectResultPrefix2) {
		return true
	}

	return false
}
