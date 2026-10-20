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

	vkId := uint(params.Message.Message.FromID)
	manager, managerErr := repository.FindOneManagerByVkId(vkId)
	if (managerErr != nil) {
		replyParams.Reply("You are not registered as a manager / 您尚未注册为管理员")
		return
	}

	code, codeErr := generateLoginCode()
	if (codeErr != nil) {
		replyParams.Reply("Ошибка генерации кода, попробуйте позже.")
		return
	}

	expiresAt := time.Now().Add(loginCodeLifetime)
	if err := repository.SetManagerLoginCode(manager, code, expiresAt); err != nil {
		replyParams.Reply("Ошибка сохранения кода, попробуйте позже.")
		return
	}

	audit.Record(manager.FractionId, manager.Id, model.AuditActionLoginRequest, "запрошена ссылка для входа")

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
