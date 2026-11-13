package messageHandler

import (
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"

	"github.com/SevereCloud/vksdk/v2/object"
)

const profileButtonLabel = "🥷 Профиль"
const webPanelButtonLabel = "🌐 Веб-панель"
const inventoryStartButtonLabel = "🎒 Анализировать инвентарь"
const inventoryFinishButtonLabel = "🏁 Завершить анализ"
const inventoryApplyButtonLabel = "✅ Провести анализ"
const inventoryContinueButtonLabel = "↩️ Продолжить сбор"
const inventoryCancelButtonLabel = "❌ Отменить анализ"

// mainKeyboard is the private chat's input-area keyboard (not an inline
// one under a message), always chosen by the sender's inventory analysis
// state so every reply shows the right buttons:
//   - no analysis: Профиль, Веб-панель, Анализировать инвентарь;
//   - collecting:  Профиль, Веб-панель, Завершить анализ;
//   - confirming:  only Провести / Продолжить сбор / Отменить.
//
// Buttons are callback-type: a tap is routed by HandleMessageEvent through
// the button's payload, not by resending the visible label as a chat
// message, so it can't be spoofed by typing the label.
func mainKeyboard(vkId int) *object.MessagesKeyboard {
	return keyboardForDraft(currentInventoryDraft(vkId))
}

func keyboardForDraft(draft *model.InventoryDraft) *object.MessagesKeyboard {
	keyboard := object.NewMessagesKeyboard(false)

	if (draft != nil && draft.Confirming) {
		keyboard.AddRow()
		keyboard.AddCallbackButton(inventoryApplyButtonLabel, map[string]string{"cmd": cmdInventoryApply}, object.Positive)
		keyboard.AddRow()
		keyboard.AddCallbackButton(inventoryContinueButtonLabel, map[string]string{"cmd": cmdInventoryContinue}, object.Secondary)
		keyboard.AddRow()
		keyboard.AddCallbackButton(inventoryCancelButtonLabel, map[string]string{"cmd": cmdInventoryCancel}, object.Negative)
		return keyboard
	}

	keyboard.AddRow()
	keyboard.AddCallbackButton(profileButtonLabel, map[string]string{"cmd": "profile"}, object.Primary)
	keyboard.AddCallbackButton(webPanelButtonLabel, map[string]string{"cmd": "login"}, object.Primary)

	keyboard.AddRow()
	if (draft != nil) {
		keyboard.AddCallbackButton(inventoryFinishButtonLabel, map[string]string{"cmd": cmdInventoryFinish}, object.Positive)
	} else {
		keyboard.AddCallbackButton(inventoryStartButtonLabel, map[string]string{"cmd": cmdInventoryStart}, object.Secondary)
	}

	return keyboard
}

func currentInventoryDraft(vkId int) *model.InventoryDraft {
	user := repository.FindOneUserByVkId(vkId)
	if (user == nil) {
		return nil
	}

	draft, err := repository.FindInventoryDraft(user.Id)
	if (err != nil) {
		return nil
	}

	return draft
}
