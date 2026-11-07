package command

import (
	"fmt"
	"strings"

	"jiqiren/bot/battletime"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/leveling"
	"jiqiren/bot/messageHandler/reply"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/events"
)

type Profile struct {
	Message *events.MessageNewObject
	Vk *api.VK
}

func (params Profile) Handle() {
	replyParams := reply.MakeParams(&params.Message.Message, params.Vk)

	vkId := uint(params.Message.Message.FromID)
	profile, profileErr := repository.FindLastProfileByVkId(vkId)
	if (profileErr != nil) {
		replyParams.Reply("У тебя пока нет сохранённого профиля — отправь его боту, чтобы посмотреть сводку.")
		return
	}

	replyParams.Reply(buildProfileSummary(*profile))
}

func buildProfileSummary(profile model.Profile) string {
	lines := make([]string, 0, 6)
	lines = append(lines, profileHeaderLine(profile))
	lines = append(lines, profileExperienceLine(profile))
	lines = append(lines, fmt.Sprintf("📡: %d; 💾: %d; 📱: %d; 🔎: %d",
		profile.Practice, profile.Theory, profile.Cunning, profile.Wisdom))

	if attack, err := repository.FindBestAttackProfile(profile.UserId); (err == nil) {
		lines = append(lines, fmt.Sprintf("АТК: ⚔️%d; 🛡️%d",
			attack.Practice*2+attack.Cunning*3, attack.Theory*2+attack.Wisdom*3))
	}

	if defense, err := repository.FindBestDefenseProfile(profile.UserId); (err == nil) {
		lines = append(lines, fmt.Sprintf("ДЕФ: ⚔️%d; 🛡️%d",
			defense.Practice*2+defense.Cunning*3, defense.Theory*2+defense.Wisdom*3))
	}

	lines = append(lines, "Актуальность: "+profile.MessageDate.In(battletime.Moscow).Format("15:04 02.01.2006"))

	return strings.Join(lines, "\n")
}

func profileHeaderLine(profile model.Profile) string {
	squadPart := ""
	if (profile.TeamId != nil) {
		if squad, err := repository.FindOneTeamById(*profile.TeamId); (err == nil) {
			squadPart = squad.Code + "|"
		}
	}

	fractionIcon := ""
	if fraction, err := repository.FindOneFractionById(profile.FractionId); (err == nil) {
		fractionIcon = fraction.Icon
	}

	levelPart := fmt.Sprintf("%d", profile.Level)
	if (profile.Prestige > 0) {
		levelPart += fmt.Sprintf("💎%d", profile.Prestige)
	}

	return fmt.Sprintf("%s%s%s%s%s%s (%s)",
		leadIcon(profile.Lead), squadPart, wayIcon(profile.Way), fractionIcon, profile.Status, profile.Username, levelPart)
}

func profileExperienceLine(profile model.Profile) string {
	progress := leveling.ProgressFor(profile.Level, uint64(profile.Experience))
	if (!progress.Ok) {
		return fmt.Sprintf("💡: %d", profile.Experience)
	}

	return fmt.Sprintf("💡: %d (%.1f%%, осталось %d)", profile.Experience, progress.Percent, progress.Remaining)
}

func leadIcon(lead uint) string {
	switch (lead) {
	case 2:
		return "🔸"
	case 1:
		return "▫"
	default:
		return ""
	}
}

func wayIcon(way uint) string {
	switch (way) {
	case 1:
		return "⬛"
	case 2:
		return "⬜"
	case 3:
		return "🔲"
	case 4:
		return "🔳"
	default:
		return ""
	}
}
