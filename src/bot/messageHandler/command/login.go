package command

import (
	"crypto/rand"
	"fmt"
	"time"

	"jiqiren/bot/audit"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/messageHandler/reply"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/events"
)

const loginCodeAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const loginCodeLength = 6
const loginCodeLifetime = 3 * time.Minute

type Login struct {
	Message *events.MessageNewObject
	Vk *api.VK
}

func (params Login) Handle() {
	replyParams := reply.MakeParams(&params.Message.Message, params.Vk)

	if (params.Message.Message.PeerID != params.Message.Message.FromID) {
		replyParams.Reply("Команда доступна только в личных сообщениях.")
		return
	}

	// The panel is open to every player who has ever sent a profile (for
	// "Мой профиль"), plus managers — who may not have sent one yet.
	vkId := uint(params.Message.Message.FromID)
	manager, managerErr := repository.FindOneManagerByVkId(vkId)
	isManager := managerErr == nil

	user := repository.FindOneUserByVkId(int(vkId))
	if (!isManager && (user == nil || !repository.UserHasProfiles(user.Id))) {
		replyParams.Reply("Сначала отправьте боту свой профиль / 请先向机器人发送您的个人资料")
		return
	}

	if (user == nil) {
		user = repository.FindOrCreateUserByVkId(int(vkId))
	}

	code, codeErr := generateLoginCode()
	if (codeErr != nil) {
		replyParams.Reply("Ошибка генерации кода, попробуйте позже.")
		return
	}

	expiresAt := time.Now().Add(loginCodeLifetime)
	if err := repository.SetUserLoginCode(user, code, expiresAt); err != nil {
		replyParams.Reply("Ошибка сохранения кода, попробуйте позже.")
		return
	}

	if (isManager) {
		audit.Record(manager.FractionId, manager.Id, model.AuditActionLoginRequest, "запрошена ссылка для входа")
	}

	replyParams.Reply(fmt.Sprintf("https://jiqiren.huoqiang.ltd/?login=%s", code))
}

func generateLoginCode() (string, error) {
	buf := make([]byte, loginCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	code := make([]byte, loginCodeLength)
	for i, b := range buf {
		code[i] = loginCodeAlphabet[int(b) % len(loginCodeAlphabet)]
	}

	return string(code), nil
}
