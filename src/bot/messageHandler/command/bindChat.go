package command

import (
	"fmt"
	"os"
	"strings"

	"jiqiren/bot/audit"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/messageHandler/reply"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/events"
)

var bindChatTriggers = []string{"bind", "привязать"}

type BindChat struct {
	Message *events.MessageNewObject
	Vk *api.VK
}

func (params BindChat) Handle() {
	replyParams := reply.MakeParams(&params.Message.Message, params.Vk)

	vkId := uint(params.Message.Message.FromID)
	manager, managerErr := repository.FindOneManagerByVkId(vkId)
	if (managerErr != nil) {
		replyParams.Reply("You are not registered as a manager / 您尚未注册为管理员")
		return
	}

	code := bindChatArgument(params.Message.Message.Text)

	squad, squadErr := resolveBindChatSquad(manager, code)
	if (squadErr != nil) {
		replyParams.Reply(squadErr.Error())
		return
	}

	peerId := params.Message.Message.PeerID
	chatId := int64(peerId)
	chatName := fetchChatTitle(params.Vk, peerId)

	if setErr := repository.SetTeamChat(squad, &chatId, chatName); (setErr != nil) {
		replyParams.Reply("Не удалось привязать чат, попробуйте позже.")
		return
	}

	audit.Record(manager.FractionId, manager.Id, model.AuditActionBindSquadChat,
		fmt.Sprintf("отряд %s: привязан чат", squad.Code))

	replyParams.Reply(fmt.Sprintf("Этот чат привязан к отряду %s.", squad.Code))
}

// resolveBindChatSquad enforces the pairing the command is meant for: a
// manager already assigned to a squad (deputy or squad lead) binds their
// own squad with no argument, while a teamlead must name the squad since
// they don't have a "own squad" to default to.
func resolveBindChatSquad(manager *model.Manager, code string) (*model.Team, error) {
	isTeamlead := manager.AccountType != nil && *manager.AccountType == model.AccountTypeTeamlead
	isSquadManager := manager.AccountType != nil &&
		(*manager.AccountType == model.AccountTypeDeputy || *manager.AccountType == model.AccountTypeSquadLead)

	if (isTeamlead) {
		if (len(code) == 0) {
			return nil, fmt.Errorf("Тимлиду нужно указать код отряда, например: !bind DD")
		}

		squad, squadErr := repository.FindOneTeamByCodeAndFraction(code, manager.FractionId)
		if (squadErr != nil) {
			return nil, fmt.Errorf("Отряд с кодом %s не найден.", code)
		}

		return squad, nil
	}

	if (isSquadManager) {
		if (len(code) > 0) {
			return nil, fmt.Errorf("Тебе не нужно указывать код отряда, команда привяжет твой собственный отряд.")
		}

		if (manager.SquadId == nil) {
			return nil, fmt.Errorf("Тебе не назначен отряд.")
		}

		squad, squadErr := repository.FindOneTeamById(*manager.SquadId)
		if (squadErr != nil) {
			return nil, fmt.Errorf("Отряд не найден.")
		}

		return squad, nil
	}

	return nil, fmt.Errorf("У тебя нет прав привязывать чат к отряду.")
}

func bindChatArgument(text string) string {
	prefix := os.Getenv("PREFIX")
	for _, trigger := range bindChatTriggers {
		withArgument := prefix + trigger + " "
		if (strings.HasPrefix(text, withArgument)) {
			return strings.TrimSpace(text[len(withArgument):])
		}
	}

	return ""
}

func fetchChatTitle(vk *api.VK, peerId int) *string {
	response, err := vk.MessagesGetConversationsByID(api.Params{"peer_ids": peerId})
	if (err != nil || len(response.Items) == 0) {
		return nil
	}

	title := response.Items[0].ChatSettings.Title
	if (title == "") {
		return nil
	}

	return &title
}
