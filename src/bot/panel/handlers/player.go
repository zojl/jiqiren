package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"jiqiren/bot/audit"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"

	"github.com/go-chi/chi/v5"
)

type playerProfileView struct {
	MessageDate string
	Username string
	StatusIcon string
	LeadIcon string
	WayIcon string
	FractionIcon string
	FractionName string
	SquadLabel string
	TransferNote string

	Level uint
	Prestige uint
	Experience uint
	ExperienceLimit uint
	ExperienceNeeded uint
	ExperiencePercent string
	HasExperienceLimit bool

	Money uint
	Vkcoin string
	Points uint
	Bitcoins uint
	Disks uint
	Pages uint
	Chips uint
	Instructions uint
	Stocks uint
	Motivation uint
	MotivationLimit uint
	Practice uint
	Theory uint
	Cunning uint
	Wisdom uint
	Stamina uint
	TargetLabel string
	SleepAt string
}

type playerStatsSnapshotView struct {
	MessageDate string
	Username string
	Practice uint
	Theory uint
	Cunning uint
	Wisdom uint
	Attack uint
	Defense uint
}

type playerPageData struct {
	VkId uint
	Profile playerProfileView
	BestAttack *playerStatsSnapshotView
	BestDefense *playerStatsSnapshotView
	BackHref string
	BackLabel string
}

// PlayerPage shows one player's last reported profile in full, plus their
// best-ever attack and defense snapshots. Visible to the fraction's
// teamlead/deputy, and to the squad lead(s) whose squad the player belongs
// to — both the natural (profile-reported) and the override-effective one,
// since those differ while a player is mid-transfer between squads.
func PlayerPage(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	vkId, parseErr := strconv.ParseUint(chi.URLParam(r, "vk_id"), 10, 64)
	if (parseErr != nil) {
		http.Error(w, "Некорректный vk id", http.StatusBadRequest)
		return
	}

	profile, profileErr := repository.FindLastProfileByVkId(uint(vkId))
	if (profileErr != nil) {
		http.Error(w, "Игрок не найден", http.StatusNotFound)
		return
	}

	users, usersErr := repository.FindUsersByIds([]uint{profile.UserId})
	if (usersErr != nil || len(users) == 0) {
		http.Error(w, "Игрок не найден", http.StatusNotFound)
		return
	}
	user := users[0]

	naturalSquadId := profile.TeamId
	effectiveSquadId := profile.EffectiveSquadId(user.SquadOverrideId)

	if (!permissions.CanViewPlayer(profile.FractionId, naturalSquadId, effectiveSquadId)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionViewPlayer,
		fmt.Sprintf("просмотр игрока %s (vk id %d)", profile.Username, vkId))

	allFractions, fractionsErr := repository.FindAllFractions()
	if (fractionsErr != nil) {
		http.Error(w, "Ошибка загрузки фракций", http.StatusInternalServerError)
		return
	}

	now := time.Now()

	view := buildPlayerProfileView(*profile, allFractions, naturalSquadId, effectiveSquadId, now)

	var bestAttack *playerStatsSnapshotView
	if attackProfile, err := repository.FindBestAttackProfile(profile.UserId); (err == nil) {
		bestAttack = buildPlayerStatsSnapshotView(*attackProfile, now)
	}

	var bestDefense *playerStatsSnapshotView
	if defenseProfile, err := repository.FindBestDefenseProfile(profile.UserId); (err == nil) {
		bestDefense = buildPlayerStatsSnapshotView(*defenseProfile, now)
	}

	backHref, backLabel := "/panel", "← назад"
	if (effectiveSquadId != nil) {
		if squad, err := repository.FindOneTeamById(*effectiveSquadId); (err == nil) {
			backHref = "/panel/squads/" + squad.Code
			backLabel = "← к отряду"
		}
	}

	templates.Render(w, r, "player_page", playerPageData{
		VkId: uint(vkId),
		Profile: view,
		BestAttack: bestAttack,
		BestDefense: bestDefense,
		BackHref: backHref,
		BackLabel: backLabel,
	})
}

func buildPlayerProfileView(profile model.Profile, allFractions []model.Fraction, naturalSquadId *uint, effectiveSquadId *uint, now time.Time) playerProfileView {
	fractionIcon, fractionName := "", ""
	for _, fraction := range allFractions {
		if (fraction.Id == profile.FractionId) {
			fractionIcon = fraction.Icon
			fractionName = fraction.Name
			break
		}
	}

	squadLabel := "—"
	if (effectiveSquadId != nil) {
		if squad, err := repository.FindOneTeamById(*effectiveSquadId); (err == nil) {
			squadLabel = squad.Code + " " + squad.Name
		}
	}

	// A transfer in progress: the profile's own squad tag disagrees with
	// the override that actually governs membership — surface the other
	// one too rather than silently picking just the effective squad.
	transferNote := ""
	if (naturalSquadId != nil && (effectiveSquadId == nil || *naturalSquadId != *effectiveSquadId)) {
		if squad, err := repository.FindOneTeamById(*naturalSquadId); (err == nil) {
			transferNote = "по профилю — " + squad.Code + " " + squad.Name
		}
	}

	view := playerProfileView{
		MessageDate: formatMskMoment(profile.MessageDate, now),
		Username: profile.Username,
		StatusIcon: profile.Status,
		LeadIcon: leadIcon(profile.Lead),
		WayIcon: wayIcon(profile.Way),
		FractionIcon: fractionIcon,
		FractionName: fractionName,
		SquadLabel: squadLabel,
		TransferNote: transferNote,

		Level: profile.Level,
		Prestige: profile.Prestige,
		Experience: profile.Experience,
		ExperienceLimit: profile.ExperienceLimit,

		Money: profile.Money,
		Vkcoin: fmt.Sprintf("%.2f", profile.Vkcoin),
		Points: profile.Points,
		Bitcoins: profile.Bitcoins,
		Disks: profile.Disks,
		Pages: profile.Pages,
		Chips: profile.Chips,
		Instructions: profile.Instructions,
		Stocks: profile.Stocks,
		Motivation: profile.Motivation,
		MotivationLimit: profile.MotivationLimit,
		Practice: profile.Practice,
		Theory: profile.Theory,
		Cunning: profile.Cunning,
		Wisdom: profile.Wisdom,
		Stamina: profile.Stamina,
		TargetLabel: targetLabel(allFractions, profile.TargetId),
		SleepAt: "—",
	}

	if (profile.SleepAt != nil) {
		view.SleepAt = formatMskMoment(*profile.SleepAt, now)
	}

	// ExperienceLimit is 0 for profiles stored before this field existed
	// (or if the message didn't carry it) — nothing to divide by then.
	if (profile.ExperienceLimit > 0) {
		view.HasExperienceLimit = true
		if (profile.Experience < profile.ExperienceLimit) {
			view.ExperienceNeeded = profile.ExperienceLimit - profile.Experience
		}
		view.ExperiencePercent = fmt.Sprintf("%.1f%%", float64(profile.Experience)/float64(profile.ExperienceLimit)*100)
	}

	return view
}

func buildPlayerStatsSnapshotView(profile model.Profile, now time.Time) *playerStatsSnapshotView {
	return &playerStatsSnapshotView{
		MessageDate: formatMskMoment(profile.MessageDate, now),
		Username: profile.Username,
		Practice: profile.Practice,
		Theory: profile.Theory,
		Cunning: profile.Cunning,
		Wisdom: profile.Wisdom,
		Attack: profile.Practice*2 + profile.Cunning*3,
		Defense: profile.Theory*2 + profile.Wisdom*3,
	}
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
