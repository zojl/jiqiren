package messageHandler

import (
	"encoding/json"
	"strings"
	"testing"

	"jiqiren/bot/database/model"
	"jiqiren/bot/inventory"
)

type keyboardButton struct {
	Label string
	Cmd string
}

// buttons flattens a keyboard into (label, cmd) rows.
func buttons(t *testing.T, draft *model.InventoryDraft) [][]keyboardButton {
	t.Helper()

	var parsed struct {
		Buttons [][]struct {
			Action struct {
				Label string `json:"label"`
				Payload string `json:"payload"`
			} `json:"action"`
		} `json:"buttons"`
	}
	if err := json.Unmarshal([]byte(keyboardForDraft(draft).ToJSON()), &parsed); (err != nil) {
		t.Fatal(err)
	}

	rows := [][]keyboardButton{}
	for _, row := range parsed.Buttons {
		buttons := []keyboardButton{}
		for _, button := range row {
			var payload struct{ Cmd string `json:"cmd"` }
			json.Unmarshal([]byte(button.Action.Payload), &payload)
			buttons = append(buttons, keyboardButton{Label: button.Action.Label, Cmd: payload.Cmd})
		}
		rows = append(rows, buttons)
	}

	return rows
}

func TestKeyboardStates(t *testing.T) {
	idle := buttons(t, nil)
	if (len(idle) != 2 || idle[0][0].Cmd != "profile" || idle[0][1].Cmd != "login" || idle[1][0].Cmd != cmdInventoryStart || idle[1][0].Label != inventoryStartButtonLabel) {
		t.Fatalf("no analysis: %+v", idle)
	}

	collecting := buttons(t, &model.InventoryDraft{})
	if (len(collecting) != 2 || collecting[0][0].Cmd != "profile" || collecting[1][0].Cmd != cmdInventoryFinish) {
		t.Fatalf("collecting: %+v", collecting)
	}

	// Confirming hides everything but the three answers.
	confirming := buttons(t, &model.InventoryDraft{Confirming: true})
	if (len(confirming) != 3 || confirming[0][0].Cmd != cmdInventoryApply || confirming[1][0].Cmd != cmdInventoryContinue || confirming[2][0].Cmd != cmdInventoryCancel) {
		t.Fatalf("confirming: %+v", confirming)
	}

	for _, rows := range [][][]keyboardButton{idle, collecting, confirming} {
		for _, row := range rows {
			for _, button := range row {
				if (button.Cmd != "profile" && button.Cmd != "login" && !isInventoryCommand(button.Cmd)) {
					t.Errorf("button %q sends a command nothing handles: %q", button.Label, button.Cmd)
				}
			}
		}
	}
}

func TestItemsCount(t *testing.T) {
	cases := map[int]string{0: "0 вещей", 1: "1 вещь", 2: "2 вещи", 4: "4 вещи", 5: "5 вещей", 11: "11 вещей", 12: "12 вещей",
		14: "14 вещей", 21: "21 вещь", 22: "22 вещи", 25: "25 вещей", 101: "101 вещь", 111: "111 вещей", 112: "112 вещей"}
	for count, want := range cases {
		if got := itemsCount(count); (got != want) {
			t.Errorf("itemsCount(%d) = %q, want %q", count, got, want)
		}
	}
}

func TestProgressText(t *testing.T) {
	summary, _ := inventory.Parse("Инвентарь:\n\n🎒Рюкзак:\n📵NetShark (⬛, +8📡)\n\n🏡Дом: 2 предмета\n🏭Склад: 8/10\n🗄Серверная: 1/1")
	storage, _ := inventory.Parse("🏭Склад:\n\n1) ⌚WatchPod (⬛, +22📡, +29💾)\n\nСтраница 1 из 3")

	got := progressText([]inventory.Forward{{Message: summary}, {Message: storage}})
	want := "Собрано:\n🎒 Рюкзак: 1\n🏡 Дом: 0 из 2\n🏭 Склад: 1 из 8 (нет стр. 2, 3)\n🗄 Серверная: 0 из 1"
	if (got != want) {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}

	withoutSummary := progressText([]inventory.Forward{{Message: storage}})
	if (!strings.Contains(withoutSummary, "🎒 Рюкзак: не собран") || !strings.Contains(withoutSummary, "🏡 Дом: 0\n")) {
		t.Fatalf("no summary — no expectations:\n%s", withoutSummary)
	}
}

func TestSplitIntoChunks(t *testing.T) {
	lines := make([]string, 0, 50)
	for i := 0; i < 50; i++ {
		lines = append(lines, strings.Repeat("я", 99))
	}
	text := strings.Join(lines, "\n")

	chunks := splitIntoChunks(text, 1000)
	if (len(chunks) != 5) {
		t.Fatalf("expected 5 chunks of 10 lines, got %d", len(chunks))
	}

	for _, chunk := range chunks {
		if (len([]rune(chunk)) > 1000 || strings.HasSuffix(chunk, "\n")) {
			t.Fatalf("chunk of %d runes", len([]rune(chunk)))
		}
	}

	if (strings.Join(chunks, "\n") != text) {
		t.Fatal("chunks must add back up to the original text")
	}

	if got := splitIntoChunks("short", 1000); (len(got) != 1 || got[0] != "short") {
		t.Fatalf("short text: %q", got)
	}

	// A single line longer than the limit isn't cut mid-line.
	long := strings.Repeat("x", 1500)
	if got := splitIntoChunks(long, 1000); (len(got) != 1 || got[0] != long) {
		t.Fatal("an overlong line stays whole")
	}
}
