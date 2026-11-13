package command

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"jiqiren/bot/audit"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/messageHandler/reply"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/events"
)

const loginCodeAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const loginCodeLength = 8
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

	replyParams.Reply(fmt.Sprintf(
		"https://jiqiren.huoqiang.ltd/?login=%s\n\nСсылка одноразовая и действует %d мин.",
		code, int(loginCodeLifetime.Minutes()),
	))
}

// generateLoginCode draws each character uniformly — a plain `byte % 36`
// would favor the first few letters of the alphabet.
func generateLoginCode() (string, error) {
	code := make([]byte, loginCodeLength)
	for i := range code {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(loginCodeAlphabet))))
		if (err != nil) {
			return "", err
		}

		code[i] = loginCodeAlphabet[index.Int64()]
	}

	return string(code), nil
}
