package messageHandler

import (
	"log"
	"strconv"
	"strings"
	"time"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/inventory"
	"jiqiren/bot/messageHandler/reply"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/object"
)

// The inventory analysis mode, private chat only. "Анализировать инвентарь"
// starts a draft; while it's open, forwarded inventory screens are
// collected into it; "Завершить анализ" shows what would be saved and asks
// to confirm; only "Провести анализ" replaces the user's stored inventory.

const (
	cmdInventoryStart = "inventory_start"
	cmdInventoryFinish = "inventory_finish"
	cmdInventoryContinue = "inventory_continue"
	cmdInventoryApply = "inventory_apply"
	cmdInventoryCancel = "inventory_cancel"
)

// VK caps a message at 4096 characters; stay well below it, since emoji
// may count double.
const maxReplyRunes = 3000

func isInventoryCommand(cmd string) bool {
	switch (cmd) {
	case cmdInventoryStart, cmdInventoryFinish, cmdInventoryContinue, cmdInventoryApply, cmdInventoryCancel:
		return true
	}

	return false
}

func handleInventoryCommand(cmd string, message *object.MessagesMessage, vk *api.VK) {
	if (message.PeerID != message.FromID) {
		return
	}

	replyParams := reply.MakeParams(message, vk)

	user := repository.FindOneUserByVkId(message.FromID)
	if (user == nil) {
		replyParams.Reply("Сначала отправьте боту свой профиль.")
		return
	}

	if (cmd == cmdInventoryStart) {
		startInventoryAnalysis(user, replyParams)
		return
	}

	draft, err := repository.FindInventoryDraft(user.Id)
	if (err != nil) {
		replyParams.ReplyWithKeyboard("Анализ инвентаря не запущен.", keyboardForDraft(nil))
		return
	}

	switch (cmd) {
	case cmdInventoryFinish:
		finishInventoryAnalysis(draft, replyParams)
	case cmdInventoryContinue:
		continueInventoryAnalysis(draft, replyParams)
	case cmdInventoryApply:
		applyInventoryAnalysis(draft, replyParams)
	case cmdInventoryCancel:
		cancelInventoryAnalysis(draft, replyParams)
	}
}

func startInventoryAnalysis(user *model.User, replyParams *reply.Params) {
	draft, err := repository.StartInventoryDraft(user.Id, time.Now())
	if (err != nil) {
		log.Printf("inventory: failed to start draft for user %d: %v", user.Id, err)
		replyParams.Reply("Не удалось начать анализ, попробуйте позже.")
		return
	}

	replyParams.ReplyWithKeyboard(
		"🎒 Анализ инвентаря начат.\n\n"+
			"Перешлите мне из игры экраны инвентаря:\n"+
			"• общий инвентарь или 🎒 рюкзак\n"+
			"• 🏡 дом, 🏭 склад и 🗄 серверную — все страницы\n\n"+
			"Можно пересылать по несколько сообщений сразу. Принимаются только сообщения, полученные после начала анализа.\n"+
			"Когда всё перешлёте, нажмите «"+inventoryFinishButtonLabel+"».",
		keyboardForDraft(draft),
	)
}

func finishInventoryAnalysis(draft *model.InventoryDraft, replyParams *reply.Params) {
	forwards, ok := loadDraftForwards(draft, replyParams)
	if (!ok) {
		return
	}

	if err := repository.SetInventoryDraftConfirming(draft, true); (err != nil) {
		replyParams.Reply("Не удалось сохранить, попробуйте позже.")
		return
	}

	if (len(forwards) == 0) {
		replyParams.ReplyWithKeyboard("Пока ничего не собрано. Перешлите экраны инвентаря или отмените анализ.", keyboardForDraft(draft))
		return
	}

	progress := inventory.Summarize(forwards)
	items := inventory.Merge(forwards)

	var text strings.Builder
	if (!progress.HasBackpack) {
		text.WriteString("⚠️ В анализе нет рюкзака — без него не получится рассчитать прибавку к характеристикам.\n")
		text.WriteString("Точно хотите провести анализ? ")
	} else {
		text.WriteString("Провести анализ инвентаря? ")
	}
	text.WriteString("Это перезапишет имеющиеся данные об инвентаре следующими данными:\n")

	for _, location := range inventory.Locations {
		lines := []string{}
		for _, item := range items {
			if (item.Location == location) {
				lines = append(lines, item.Line())
			}
		}

		text.WriteString("\n")
		if (len(lines) == 0) {
			text.WriteString(location.Label() + ": —\n")
			continue
		}

		text.WriteString(location.Label() + " (" + strconv.Itoa(len(lines)) + "):\n")
		text.WriteString(strings.Join(lines, "\n"))
		text.WriteString("\n")
	}

	replyInChunks(replyParams, text.String(), keyboardForDraft(draft))
}

func continueInventoryAnalysis(draft *model.InventoryDraft, replyParams *reply.Params) {
	if err := repository.SetInventoryDraftConfirming(draft, false); (err != nil) {
		replyParams.Reply("Не удалось сохранить, попробуйте позже.")
		return
	}

	forwards, ok := loadDraftForwards(draft, replyParams)
	if (!ok) {
		return
	}

	replyParams.ReplyWithKeyboard(
		"↩️ Продолжаем сбор. Пересылайте экраны инвентаря, а когда закончите — «"+inventoryFinishButtonLabel+"».\n\n"+progressText(forwards),
		keyboardForDraft(draft),
	)
}

func applyInventoryAnalysis(draft *model.InventoryDraft, replyParams *reply.Params) {
	forwards, ok := loadDraftForwards(draft, replyParams)
	if (!ok) {
		return
	}

	if (len(forwards) == 0) {
		replyParams.ReplyWithKeyboard("Нечего анализировать: ничего не собрано. Перешлите экраны инвентаря или отмените анализ.", keyboardForDraft(draft))
		return
	}

	items := inventory.Merge(forwards)
	stored := make([]model.InventoryItem, 0, len(items))
	for _, item := range items {
		stored = append(stored, inventory.ToModel(draft.UserId, item))
	}

	if err := repository.ApplyInventoryDraft(draft, stored, inventory.Summarize(forwards).HasBackpack, time.Now()); (err != nil) {
		log.Printf("inventory: failed to apply draft %d: %v", draft.Id, err)
		replyParams.Reply("Не удалось сохранить инвентарь, попробуйте позже.")
		return
	}

	replyParams.ReplyWithKeyboard(
		"✅ Инвентарь сохранён: "+itemsCount(len(items))+".\n"+
			"Лучшие наборы для атаки и защиты — в веб-панели: «Мой профиль» → «Инвентарь».",
		keyboardForDraft(nil),
	)
}

func cancelInventoryAnalysis(draft *model.InventoryDraft, replyParams *reply.Params) {
	if err := repository.DeleteInventoryDraft(draft); (err != nil) {
		replyParams.Reply("Не удалось отменить анализ, попробуйте позже.")
		return
	}

	replyParams.ReplyWithKeyboard("❌ Анализ отменён, данные инвентаря не изменились.", keyboardForDraft(nil))
}

// handleInventoryForwards collects every inventory screen forwarded in one
// chat message into the sender's draft and answers once for all of them;
// unrecognized counts the game messages in it that weren't any known
// screen, to say so in the same reply.
func handleInventoryForwards(forwards []object.MessagesMessage, unrecognized int, parentMessage *object.MessagesMessage, vk *api.VK) {
	if (parentMessage.PeerID != parentMessage.FromID) {
		return
	}

	replyParams := reply.MakeParams(parentMessage, vk)

	draft := currentInventoryDraft(parentMessage.FromID)
	if (draft == nil) {
		replyParams.ReplyWithKeyboard(
			"Чтобы проанализировать инвентарь, нажмите «"+inventoryStartButtonLabel+"», а затем перешлите экраны инвентаря.",
			mainKeyboard(parentMessage.FromID),
		)
		return
	}

	accepted := []string{}
	rejectedStale := 0
	for _, forward := range forwards {
		sentAt := time.Unix(int64(forward.Date), 0)
		if (sentAt.Before(draft.StartedAt)) {
			rejectedStale++
			continue
		}

		message, ok := inventory.Parse(forward.Text)
		if (!ok) {
			continue
		}

		err := repository.SaveInventoryDraftMessage(&model.InventoryDraftMessage{
			DraftId: draft.Id,
			Kind: string(message.Kind),
			Page: uint(message.Page),
			Text: forward.Text,
			SentAt: sentAt,
		})
		if (err != nil) {
			log.Printf("inventory: failed to save draft message for draft %d: %v", draft.Id, err)
			replyParams.Reply("Не удалось сохранить, попробуйте позже.")
			return
		}

		label := "✔ " + message.Kind.Label()
		if (message.TotalPages > 1) {
			label += ", стр. " + strconv.Itoa(message.Page) + " из " + strconv.Itoa(message.TotalPages)
		}
		accepted = append(accepted, label+" — "+itemsCount(len(message.Items)))
	}

	// New data makes a confirmation already on screen stale — back to
	// collecting, so it gets reviewed again.
	if (draft.Confirming && len(accepted) > 0) {
		if err := repository.SetInventoryDraftConfirming(draft, false); (err != nil) {
			log.Printf("inventory: failed to reset confirming for draft %d: %v", draft.Id, err)
		}
	}

	collected, ok := loadDraftForwards(draft, replyParams)
	if (!ok) {
		return
	}

	lines := accepted
	if (unrecognized > 0) {
		lines = append(lines, "✖ Не распознано сообщений: "+strconv.Itoa(unrecognized)+" — это не экраны инвентаря. Пересылайте общий инвентарь, рюкзак, дом, склад или серверную.")
	}

	if (rejectedStale > 0) {
		lines = append(lines, "✖ Не принято сообщений: "+strconv.Itoa(rejectedStale)+" — они получены до начала анализа. Откройте этот экран в игре заново и перешлите.")
	}

	replyParams.ReplyWithKeyboard(strings.Join(lines, "\n")+"\n\n"+progressText(collected), keyboardForDraft(draft))
}

func loadDraftForwards(draft *model.InventoryDraft, replyParams *reply.Params) ([]inventory.Forward, bool) {
	messages, err := repository.FindInventoryDraftMessages(draft.Id)
	if (err != nil) {
		replyParams.Reply("Не удалось загрузить собранные данные, попробуйте позже.")
		return nil, false
	}

	return inventory.ForwardsFromDraft(messages), true
}

// progressText: "Собрано:" plus one line per location — the game's own
// item count, out of what the summary screen said to expect, plus any
// pages still missing.
func progressText(forwards []inventory.Forward) string {
	progress := inventory.Summarize(forwards)

	lines := []string{"Собрано:"}
	for _, location := range inventory.Locations {
		line := location.Label() + ": "
		if (location == inventory.LocationBackpack && !progress.HasBackpack) {
			lines = append(lines, line+"не собран")
			continue
		}

		line += strconv.Itoa(progress.Count[location])
		if expected, known := progress.Expected[location]; (known && expected >= 0) {
			line += " из " + strconv.Itoa(expected)
		}

		if missing := progress.MissingPages[location]; (len(missing) > 0) {
			pages := make([]string, 0, len(missing))
			for _, page := range missing {
				pages = append(pages, strconv.Itoa(page))
			}
			line += " (нет стр. " + strings.Join(pages, ", ") + ")"
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// replyInChunks sends a possibly long text as several messages split on
// line boundaries, with the keyboard on the last one.
func replyInChunks(replyParams *reply.Params, text string, keyboard *object.MessagesKeyboard) {
	chunks := []string{}
	var current strings.Builder
	currentRunes := 0

	for _, line := range strings.Split(text, "\n") {
		lineRunes := len([]rune(line)) + 1
		if (currentRunes > 0 && currentRunes+lineRunes > maxReplyRunes) {
			chunks = append(chunks, strings.TrimRight(current.String(), "\n"))
			current.Reset()
			currentRunes = 0
		}

		current.WriteString(line)
		current.WriteString("\n")
		currentRunes += lineRunes
	}

	if (currentRunes > 0) {
		chunks = append(chunks, strings.TrimRight(current.String(), "\n"))
	}

	for i, chunk := range chunks {
		if (i == len(chunks)-1) {
			replyParams.ReplyWithKeyboard(chunk, keyboard)
		} else {
			replyParams.Reply(chunk)
		}
	}
}

// itemsCount renders "1 вещь" / "2 вещи" / "5 вещей".
func itemsCount(count int) string {
	word := "вещей"
	if (count%10 == 1 && count%100 != 11) {
		word = "вещь"
	} else if (count%10 >= 2 && count%10 <= 4 && (count%100 < 12 || count%100 > 14)) {
		word = "вещи"
	}

	return strconv.Itoa(count) + " " + word
}
