// Package inventory parses the game's inventory messages (forwarded to the
// bot during an inventory analysis), merges them into one item set, and
// picks the best attack/defense builds from it. It's pure — no database
// access — so the bot flow and the panel share it.
package inventory

import (
	"regexp"
	"strconv"
	"strings"
)

// Kind is which game screen a forwarded message comes from.
type Kind string

const (
	// KindSummary is the general "active inventory" screen: equipped items
	// one per line (stats only), plus counts for home/storage/server.
	KindSummary Kind = "summary"
	// KindBackpack is the detailed backpack — the same equipped items,
	// with firmware, price and description.
	KindBackpack Kind = "backpack"
	KindHome Kind = "home"
	KindStorage Kind = "storage"
	KindServer Kind = "server"
)

// Location is where an item lies. Both KindSummary and KindBackpack
// describe LocationBackpack — the equipped set.
type Location string

const (
	LocationBackpack Location = "backpack"
	LocationHome Location = "home"
	LocationStorage Location = "storage"
	LocationServer Location = "server"
)

// Locations in display order.
var Locations = []Location{LocationBackpack, LocationHome, LocationStorage, LocationServer}

var locationLabels = map[Location]string{
	LocationBackpack: "🎒 Рюкзак",
	LocationHome: "🏡 Дом",
	LocationStorage: "🏭 Склад",
	LocationServer: "🗄 Серверная",
}

func (location Location) Label() string {
	return locationLabels[location]
}

func (kind Kind) Location() Location {
	switch (kind) {
	case KindHome:
		return LocationHome
	case KindStorage:
		return LocationStorage
	case KindServer:
		return LocationServer
	default:
		return LocationBackpack
	}
}

var kindLabels = map[Kind]string{
	KindSummary: "🎒 инвентарь (общий)",
	KindBackpack: "🎒 рюкзак",
	KindHome: "🏡 дом",
	KindStorage: "🏭 склад",
	KindServer: "🗄 серверная",
}

func (kind Kind) Label() string {
	return kindLabels[kind]
}

// Item classes — at most one of each can be worn. Order is display order.
type Class struct {
	Icon string
	Name string
}

var Classes = []Class{
	{Icon: "🖥", Name: "Комп"},
	{Icon: "💻", Name: "Ноут"},
	{Icon: "🔀", Name: "VPN"},
	{Icon: "⌚", Name: "Часы"},
	{Icon: "📳", Name: "Смартфон"},
	{Icon: "📵", Name: "Файрволл"},
}

var rarityIcons = []string{"▫", "▪", "🔸", "🔹"}
var serverMarks = []string{"👨🏻‍💻", "👨‍💻"}

const WayBlack = "⬛"
const WayWhite = "⬜"

// Item is one parsed inventory item. The identity fields (see key) are
// what makes two items "the same item"; the rest is detail that a more
// thorough screen may add (firmware, price, description...).
type Item struct {
	Location Location

	Rarity string
	IsServer bool
	Class string
	Name string
	Way string
	Practice int
	Theory int
	Cunning int
	Wisdom int

	// Firmware (🔰) — its own stat changes aren't shown by the game, so
	// it's informational only.
	FirmwareRarity string
	FirmwareName string
	RequiredLevel uint
	Price uint
	Unsellable bool
	// Counters are the " - Побед в битвах: 0" style lines, kept verbatim.
	Counters []string
	Description string
	// Raw is the item's original text block, kept so items can be
	// re-parsed later as the format becomes better understood.
	Raw string
}

func (item Item) AttackScore() int {
	return item.Practice*2 + item.Cunning*3
}

func (item Item) DefenseScore() int {
	return item.Theory*2 + item.Wisdom*3
}

// Title is the item's identity as the game prints it, without stats:
// "🔹👨🏻‍💻🔀WireGuard сервер".
func (item Item) Title() string {
	title := item.Rarity
	if (item.IsServer) {
		title += serverMarks[0]
	}

	return title + item.Class + item.Name
}

// StatsLabel renders the parenthesized part: "⬛, +8📡, +6💾, -1📱".
func (item Item) StatsLabel() string {
	parts := []string{}
	if (item.Way != "") {
		parts = append(parts, item.Way)
	}

	for _, stat := range []struct {
		value int
		icon string
	}{{item.Practice, "📡"}, {item.Theory, "💾"}, {item.Cunning, "📱"}, {item.Wisdom, "🔎"}} {
		if (stat.value != 0) {
			parts = append(parts, signed(stat.value)+stat.icon)
		}
	}

	return strings.Join(parts, ", ")
}

// Firmware renders "🔸Отличная", or "" when unknown.
func (item Item) Firmware() string {
	return item.FirmwareRarity + item.FirmwareName
}

// Line is a one-line description for chat: title, stats and firmware.
func (item Item) Line() string {
	line := item.Title() + " (" + item.StatsLabel() + ")"
	if (item.FirmwareName != "") {
		line += " 🔰" + item.Firmware()
	}

	return line
}

func signed(value int) string {
	if (value > 0) {
		return "+" + strconv.Itoa(value)
	}

	return strconv.Itoa(value)
}

// key is the item's identity: everything except the detail fields.
func (item Item) key() string {
	server := "0"
	if (item.IsServer) {
		server = "1"
	}

	return strings.Join([]string{
		item.Rarity, server, item.Class, item.Name, item.Way,
		strconv.Itoa(item.Practice), strconv.Itoa(item.Theory), strconv.Itoa(item.Cunning), strconv.Itoa(item.Wisdom),
	}, "|")
}

// Message is one parsed forwarded message.
type Message struct {
	Kind Kind
	Page int
	TotalPages int
	Items []Item
	// Expected item counts per location, from a KindSummary message only
	// (its "🏡Дом: 2 предмета" / "🏭Склад: 8/10" lines); -1 when unknown.
	ExpectedHome int
	ExpectedStorage int
	ExpectedServer int
}

var (
	summaryHomeRe = regexp.MustCompile(`(?m)^🏡Дом: (\d+)`)
	summaryStorageRe = regexp.MustCompile(`(?m)^🏭Склад: (\d+)`)
	summaryServerRe = regexp.MustCompile(`(?m)^🗄Серверная: (\d+)`)
	pageRe = regexp.MustCompile(`(?m)^Страница (\d+) из (\d+)`)
	positionRe = regexp.MustCompile(`^\d+\)\s*`)
	statRe = regexp.MustCompile(`^([+-]?\d+)(📡|💾|📱|🔎)$`)
	priceInlineRe = regexp.MustCompile(`^(\d+)💵$`)
	firmwareRe = regexp.MustCompile(`^🔰:\s*(🔺|🔻|🔸|🔹)?\s*(.+)$`)
	counterRe = regexp.MustCompile(`^-\s*(.+?):\s*(\d+)$`)
	requiredLevelRe = regexp.MustCompile(`^Требуется\s+(\d+)`)
	priceRe = regexp.MustCompile(`^Цена:\s*(\d+)💵`)
)

// normalize drops emoji variation selectors (U+FE0F), which the game and
// VK clients add inconsistently ("⌚" vs "⌚️"), and normalizes newlines.
func normalize(text string) string {
	text = strings.ReplaceAll(text, "️", "")
	return strings.ReplaceAll(text, "\r\n", "\n")
}

// DetectKind tells whether text is one of the inventory screens.
func DetectKind(text string) (Kind, bool) {
	text = strings.TrimSpace(normalize(text))

	switch {
	case strings.HasPrefix(text, "🎒Рюкзак:"):
		if (summaryHomeRe.MatchString(text)) {
			return KindSummary, true
		}
		return KindBackpack, true
	case strings.HasPrefix(text, "🏡Дом:"):
		return KindHome, true
	case strings.HasPrefix(text, "🏭Склад:"):
		return KindStorage, true
	case strings.HasPrefix(text, "🗄Серверная:"):
		return KindServer, true
	}

	return "", false
}

// Parse parses an inventory message; ok is false if it isn't one.
func Parse(text string) (Message, bool) {
	kind, ok := DetectKind(text)
	if (!ok) {
		return Message{}, false
	}

	text = normalize(text)
	message := Message{Kind: kind, Page: 1, TotalPages: 1, ExpectedHome: -1, ExpectedStorage: -1, ExpectedServer: -1}

	if match := pageRe.FindStringSubmatch(text); (match != nil) {
		message.Page, _ = strconv.Atoi(match[1])
		message.TotalPages, _ = strconv.Atoi(match[2])
	}

	if (kind == KindSummary) {
		message.ExpectedHome = submatchInt(summaryHomeRe, text)
		message.ExpectedStorage = submatchInt(summaryStorageRe, text)
		message.ExpectedServer = submatchInt(summaryServerRe, text)
	}

	message.Items = parseItems(text, kind.Location())

	return message, true
}

func submatchInt(re *regexp.Regexp, text string) int {
	match := re.FindStringSubmatch(text)
	if (match == nil) {
		return -1
	}

	value, err := strconv.Atoi(match[1])
	if (err != nil) {
		return -1
	}

	return value
}

// parseItems walks the message line by line: a header line starts an
// item, the non-blank lines right after it are its details, and a blank
// line ends it. Lines outside any item (titles, counts, footers) are
// skipped.
func parseItems(text string, location Location) []Item {
	lines := joinWrappedHeaders(strings.Split(text, "\n"))

	items := []Item{}
	var current *Item
	var raw []string

	flush := func() {
		if (current != nil) {
			current.Raw = strings.Join(raw, "\n")
			current.Description = strings.TrimSpace(current.Description)
			items = append(items, *current)
		}
		current = nil
		raw = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if item, ok := parseHeader(trimmed); (ok) {
			flush()
			item.Location = location
			current = &item
			raw = []string{trimmed}
			continue
		}

		if (trimmed == "" || strings.HasPrefix(trimmed, "Страница ") || strings.HasPrefix(trimmed, "Ты можешь оставить")) {
			flush()
			continue
		}

		if (current == nil) {
			continue
		}

		raw = append(raw, trimmed)
		applyDetail(current, trimmed)
	}

	flush()

	return items
}

// joinWrappedHeaders glues a header whose "(stats)" part was wrapped onto
// the next line back into one line.
func joinWrappedHeaders(lines []string) []string {
	joined := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if (i+1 < len(lines) && !strings.Contains(line, "(") && startsWithClass(line)) {
			next := strings.TrimSpace(lines[i+1])
			if (strings.HasPrefix(next, "(") && strings.HasSuffix(next, ")")) {
				joined = append(joined, line+" "+next)
				i++
				continue
			}
		}
		joined = append(joined, lines[i])
	}

	return joined
}

func startsWithClass(line string) bool {
	line = positionRe.ReplaceAllString(line, "")
	line, _ = trimAnyPrefix(line, rarityIcons)
	line, _ = trimAnyPrefix(line, serverMarks)
	_, ok := trimAnyPrefix(line, classIcons())
	return ok
}

func classIcons() []string {
	icons := make([]string, 0, len(Classes))
	for _, class := range Classes {
		icons = append(icons, class.Icon)
	}

	return icons
}

func trimAnyPrefix(text string, prefixes []string) (string, bool) {
	for _, prefix := range prefixes {
		if (strings.HasPrefix(text, prefix)) {
			return strings.TrimPrefix(text, prefix), true
		}
	}

	return text, false
}

func matchedPrefix(text string, prefixes []string) string {
	for _, prefix := range prefixes {
		if (strings.HasPrefix(text, prefix)) {
			return prefix
		}
	}

	return ""
}

// parseHeader parses "[N) ][rarity][👨🏻‍💻]class Name (stats)".
func parseHeader(line string) (Item, bool) {
	item := Item{}
	rest := positionRe.ReplaceAllString(line, "")

	item.Rarity = matchedPrefix(rest, rarityIcons)
	rest = strings.TrimPrefix(rest, item.Rarity)

	if server := matchedPrefix(rest, serverMarks); (server != "") {
		item.IsServer = true
		rest = strings.TrimPrefix(rest, server)
	}

	item.Class = matchedPrefix(rest, classIcons())
	if (item.Class == "") {
		return Item{}, false
	}
	rest = strings.TrimPrefix(rest, item.Class)

	if (!strings.HasSuffix(rest, ")")) {
		return Item{}, false
	}

	open := strings.LastIndex(rest, "(")
	if (open <= 0) {
		return Item{}, false
	}

	item.Name = strings.TrimSpace(rest[:open])
	if (item.Name == "") {
		return Item{}, false
	}

	hasStat := false
	for _, part := range strings.Split(rest[open+1:len(rest)-1], ",") {
		part = strings.TrimSpace(part)
		switch {
		case part == WayBlack || part == WayWhite:
			item.Way = part
		case part == "❌":
			item.Unsellable = true
		case priceInlineRe.MatchString(part):
			price, _ := strconv.Atoi(priceInlineRe.FindStringSubmatch(part)[1])
			item.Price = uint(price)
		case statRe.MatchString(part):
			match := statRe.FindStringSubmatch(part)
			value, _ := strconv.Atoi(match[1])
			hasStat = true
			switch (match[2]) {
			case "📡":
				item.Practice = value
			case "💾":
				item.Theory = value
			case "📱":
				item.Cunning = value
			case "🔎":
				item.Wisdom = value
			}
		}
	}

	if (!hasStat) {
		return Item{}, false
	}

	return item, true
}

func applyDetail(item *Item, line string) {
	switch {
	case firmwareRe.MatchString(line):
		match := firmwareRe.FindStringSubmatch(line)
		item.FirmwareRarity = match[1]
		item.FirmwareName = strings.TrimSpace(match[2])
	case counterRe.MatchString(line):
		item.Counters = append(item.Counters, strings.TrimSpace(strings.TrimPrefix(line, "-")))
	case requiredLevelRe.MatchString(line):
		level, _ := strconv.Atoi(requiredLevelRe.FindStringSubmatch(line)[1])
		item.RequiredLevel = uint(level)
	case priceRe.MatchString(line):
		price, _ := strconv.Atoi(priceRe.FindStringSubmatch(line)[1])
		item.Price = uint(price)
	case strings.HasPrefix(line, "❌"):
		item.Unsellable = true
	default:
		if (item.Description != "") {
			item.Description += "\n"
		}
		item.Description += line
	}
}
