package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"jiqiren/bot/audit"
	"jiqiren/bot/battletime"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"
	"jiqiren/bot/readiness"

	"github.com/go-chi/chi/v5"
)

var moscow = battletime.Moscow

const inactiveStaleAfter = 7 * 24 * time.Hour
const inactiveHiddenAfter = 30 * 24 * time.Hour

type squadMemberView struct {
	Name string
	VkId uint
	WayIcon string
	Stale bool
	LastSeen string

	Level uint
	Practice uint
	Theory uint
	Cunning uint
	Wisdom uint
	Attack uint
	Defense uint

	Stamina uint
	StaminaLabel string
	StaminaClass string

	Money uint
	MoneyLabel string
	MoneyClass string

	TargetLabel string
	TargetClass string

	SleepAt string
	SleepWarning bool

	HasReport bool
	ReportStamina uint
	ReportMoney string

	IsOverride bool
}

type squadHiddenMemberView struct {
	Name string
	VkId uint
	Info string
	IsOverride bool
}

type squadFormData struct {
	Action string
	Name string
	ShowCodeFields bool
}

type squadPageData struct {
	Id uint
	Name string
	Code string
	ChatName string
	CanUnbindChat bool
	TargetLabel string
	CurrentTargetIcon string
	TargetOptions []targetIconOption
	HasToken bool
	CanSetTarget bool
	CanRename bool
	CanSetToken bool
	CanManageSchedule bool
	CanManageMembers bool
	ShowBackLink bool
	Members []squadMemberView
	HiddenMembers []squadHiddenMemberView
}

func SquadPage(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, squadErr := repository.FindOneTeamByCodeAndFraction(chi.URLParam(r, "code"), permissions.FractionId())
	if (squadErr != nil) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if (!permissions.CanViewSquad(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	allFractions, fractionsErr := repository.FindAllFractions()
	if (fractionsErr != nil) {
		http.Error(w, "Ошибка загрузки фракций", http.StatusInternalServerError)
		return
	}

	members, hiddenMembers, membersErr := buildSquadMembers(squad, allFractions)
	if (membersErr != nil) {
		http.Error(w, "Не удалось загрузить участников", http.StatusInternalServerError)
		return
	}

	target := model.TargetNone
	if (squad.Target != nil) {
		target = *squad.Target
	}

	homeSquadId, isHome := permissions.HomeSquadId()

	templates.Render(w, "squad_page", squadPageData{
		Id: squad.Id,
		Name: squad.Name,
		Code: squad.Code,
		ChatName: squadChatLabel(squad),
		CanUnbindChat: squad.ChatId != nil && permissions.CanRenameSquad(squad.Id),
		TargetLabel: targetLabel(allFractions, squad.Target),
		CurrentTargetIcon: targetIcon(allFractions, squad.Target),
		TargetOptions: buildTargetIconOptions(allFractions, squad.FractionId, target),
		HasToken: squad.TargetToken != "",
		CanSetTarget: permissions.CanSetSquadTarget(squad.Id),
		CanRename: permissions.CanRenameSquad(squad.Id),
		CanSetToken: permissions.CanSetSquadToken(squad.Id),
		CanManageSchedule: permissions.CanManageSchedule(squad.Id),
		CanManageMembers: permissions.CanManageMembers(squad.Id),
		ShowBackLink: !isHome || homeSquadId != squad.Id,
		Members: members,
		HiddenMembers: hiddenMembers,
	})
}

// buildSquadMembers derives the squad roster from each player's own most
// recently reported profile — there is no authoritative membership list.
// Effective squad is TeamId unless the player has a manual override
// (model.Profile.EffectiveSquadId), which always wins. Besides the main
// roster, it separately returns two kinds of "hidden" members whose
// natural TeamId is this squad: ones overridden elsewhere (reversible —
// shown so they can be brought back) and ones silent for a month (shown
// so it's clear where they went, not just silently dropped).
func buildSquadMembers(squad *model.Team, allFractions []model.Fraction) ([]squadMemberView, []squadHiddenMemberView, error) {
	profiles, err := repository.FindLatestProfilesForTeam(squad.Id)
	if (err != nil) {
		return nil, nil, err
	}

	userIds := make([]uint, 0, len(profiles))
	for _, profile := range profiles {
		userIds = append(userIds, profile.UserId)
	}

	users, _ := repository.FindUsersByIds(userIds)
	vkIdByUserId := make(map[uint]uint, len(users))
	overrideByUserId := make(map[uint]*uint, len(users))
	for _, user := range users {
		vkIdByUserId[user.Id] = user.VkId
		overrideByUserId[user.Id] = user.SquadOverrideId
	}

	now := time.Now()
	nextBattle := battletime.Next(now)
	lastBattle := nextBattle.Add(-24 * time.Hour)
	sleepWarningStart := nextBattle.Add(-12 * time.Hour)

	lastBattleMsk := lastBattle.In(moscow)
	lastBattleDate := time.Date(lastBattleMsk.Year(), lastBattleMsk.Month(), lastBattleMsk.Day(), 0, 0, 0, 0, time.UTC)
	reportByUserId := make(map[uint]model.Report, len(profiles))
	if reports, reportsErr := repository.FindReportsByUserIdsAndDate(userIds, &lastBattleDate); (reportsErr == nil) {
		for _, report := range reports {
			// Guard against a report whose battle_date happens to match by
			// coincidence but whose message predates the battle itself.
			if (report.MessageDate.After(lastBattle)) {
				reportByUserId[report.UserId] = report
			}
		}
	}

	members := make([]squadMemberView, 0, len(profiles))
	hidden := make([]squadHiddenMemberView, 0)
	for _, profile := range profiles {
		override := overrideByUserId[profile.UserId]
		effectiveSquadId := profile.EffectiveSquadId(override)
		isEffective := effectiveSquadId != nil && *effectiveSquadId == squad.Id
		isNatural := profile.TeamId != nil && *profile.TeamId == squad.Id

		inactiveFor := now.Sub(profile.MessageDate)
		silentLong := inactiveFor > inactiveHiddenAfter

		if (!isEffective || silentLong) {
			if (!isNatural) {
				continue // neither naturally nor effectively this squad's — irrelevant
			}

			if (override != nil && *override != squad.Id) {
				hidden = append(hidden, squadHiddenMemberView{
					Name: profile.Username,
					VkId: vkIdByUserId[profile.UserId],
					Info: "перекинут в " + overrideSquadLabel(*override),
					IsOverride: true,
				})
			} else if (silentLong) {
				hidden = append(hidden, squadHiddenMemberView{
					Name: profile.Username,
					VkId: vkIdByUserId[profile.UserId],
					Info: fmt.Sprintf("не активен %d дн.", int(inactiveFor.Hours()/24)),
				})
			}

			continue
		}

		// A profile sent before the last battle says nothing about
		// readiness for the next one — grey out the battle-relevant
		// fields instead of showing a stale good/bad verdict.
		beforeLastBattle := profile.MessageDate.Before(lastBattle)

		staminaLabel := staminaRating(profile.Stamina)
		moneyLabel := moneyRating(profile.Money)
		staminaClass := ratingClass(staminaLabel)
		moneyClass := ratingClass(moneyLabel)
		targetClass := "rating-bad"
		if (readiness.TargetMatches(profile, squad)) {
			targetClass = "rating-good"
		}

		if (beforeLastBattle) {
			staminaClass = "rating-stale"
			moneyClass = "rating-stale"
			targetClass = "rating-stale"
		}

		member := squadMemberView{
			Name: profile.Username,
			VkId: vkIdByUserId[profile.UserId],
			WayIcon: wayIcon(profile.Way),
			Stale: inactiveFor > inactiveStaleAfter,
			LastSeen: formatMskMoment(profile.MessageDate, now),
			Level: profile.Level,
			Practice: profile.Practice,
			Theory: profile.Theory,
			Cunning: profile.Cunning,
			Wisdom: profile.Wisdom,
			Attack: profile.Practice*3 + profile.Cunning*2,
			Defense: profile.Theory*3 + profile.Wisdom*2,
			Stamina: profile.Stamina,
			StaminaLabel: staminaLabel,
			StaminaClass: staminaClass,
			Money: profile.Money,
			MoneyLabel: moneyLabel,
			MoneyClass: moneyClass,
			TargetLabel: playerTargetLabel(profile, allFractions, squad),
			TargetClass: targetClass,
			SleepAt: "—",
			IsOverride: !isNatural,
		}

		if (profile.SleepAt != nil) {
			member.SleepAt = formatMskMoment(*profile.SleepAt, now)
			member.SleepWarning = !profile.SleepAt.Before(sleepWarningStart) && !profile.SleepAt.After(nextBattle)
		}

		if report, hasReport := reportByUserId[profile.UserId]; (hasReport) {
			member.HasReport = true
			member.ReportStamina = report.Stamina
			member.ReportMoney = formatReportMoney(report)
		}

		members = append(members, member)
	}

	sort.Slice(members, func(i, j int) bool { return members[i].Name < members[j].Name })
	sort.Slice(hidden, func(i, j int) bool { return hidden[i].Name < hidden[j].Name })

	return members, hidden, nil
}

func overrideSquadLabel(squadId uint) string {
	squad, err := repository.FindOneTeamById(squadId)
	if (err != nil) {
		return fmt.Sprintf("#%d", squadId)
	}

	return squad.Code + " " + squad.Name
}

// formatReportMoney renders the battle's money delta as labeled parts —
// income, intercepted-transaction bonus, loss from a breached defense, and
// loss from a failed attack — omitting whichever parts were zero.
func formatReportMoney(report model.Report) string {
	parts := make([]string, 0, 4)
	if (report.RewardMoney > 0) {
		parts = append(parts, fmt.Sprintf("доход +%d", report.RewardMoney))
	}
	if (report.TransactionMoney > 0) {
		parts = append(parts, fmt.Sprintf("транза +%d", report.TransactionMoney))
	}
	if (report.LostMoney > 0) {
		parts = append(parts, fmt.Sprintf("пробив -%d", report.LostMoney))
	}
	if (report.AttackLostMoney > 0) {
		parts = append(parts, fmt.Sprintf("атака -%d", report.AttackLostMoney))
	}

	if (len(parts) == 0) {
		return "0"
	}

	return strings.Join(parts, ", ")
}

// formatMskMoment omits the date only when it's today — otherwise "15:04"
// alone reads as "tonight"/"just now" even when the moment is far off.
func formatMskMoment(moment time.Time, now time.Time) string {
	sleepMsk := moment.In(moscow)
	nowMsk := now.In(moscow)
	if (sleepMsk.Year() == nowMsk.Year() && sleepMsk.YearDay() == nowMsk.YearDay()) {
		return sleepMsk.Format("15:04")
	}

	return sleepMsk.Format("02.01 15:04")
}

func staminaRating(stamina uint) string {
	switch {
	case stamina >= 250:
		return "отлично"
	case stamina >= 200:
		return "хорошо"
	default:
		return "плохо"
	}
}

func moneyRating(money uint) string {
	switch {
	case money <= 9:
		return "отлично"
	case money <= 100:
		return "приемлемо"
	default:
		return "плохо"
	}
}

func ratingClass(label string) string {
	switch (label) {
	case "отлично", "хорошо":
		return "rating-good"
	case "приемлемо":
		return "rating-warn"
	default:
		return "rating-bad"
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


// squadChatLabel falls back to the raw chat id when a chat is bound but
// its title wasn't captured (e.g. bound before this field existed).
func squadChatLabel(squad *model.Team) string {
	if (squad.ChatId == nil) {
		return ""
	}

	if (squad.ChatName != nil && *squad.ChatName != "") {
		return *squad.ChatName
	}

	return fmt.Sprintf("#%d", *squad.ChatId)
}

func playerTargetLabel(profile model.Profile, allFractions []model.Fraction, squad *model.Team) string {
	if (profile.TargetId == nil) {
		return "—"
	}

	if (*profile.TargetId == squad.FractionId) {
		return "🔐 защита"
	}

	for _, fraction := range allFractions {
		if (fraction.Id == *profile.TargetId) {
			return fraction.Icon + " " + fraction.Name
		}
	}

	return "?"
}

func NewSquadForm(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanManageSquads()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	templates.Render(w, "squad_form", squadFormData{Action: "/panel/squads", ShowCodeFields: true})
}

func CreateSquad(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanManageSquads()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	squad := model.Team{
		Name: r.FormValue("name"),
		Code: r.FormValue("code"),
		FractionId: permissions.FractionId(),
	}

	if chatIdRaw := r.FormValue("chat_id"); chatIdRaw != "" {
		if chatId, parseErr := strconv.ParseInt(chatIdRaw, 10, 64); parseErr == nil {
			squad.ChatId = &chatId
		}
	}

	if createErr := repository.CreateTeam(&squad); createErr != nil {
		http.Error(w, "Не удалось создать отряд", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionCreateSquad,
		fmt.Sprintf("создан отряд %s «%s»", squad.Code, squad.Name))
	http.Redirect(w, r, "/panel", http.StatusFound)
}

func RenameSquadForm(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, squadErr := repository.FindOneTeamByCodeAndFraction(chi.URLParam(r, "code"), permissions.FractionId())
	if (squadErr != nil) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if (!permissions.CanRenameSquad(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	templates.Render(w, "squad_form", squadFormData{
		Action: "/panel/squads/" + squad.Code + "/rename",
		Name: squad.Name,
	})
}

func RenameSquad(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, squadErr := repository.FindOneTeamByCodeAndFraction(chi.URLParam(r, "code"), permissions.FractionId())
	if (squadErr != nil) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if (!permissions.CanRenameSquad(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	oldName := squad.Name
	newName := r.FormValue("name")
	if renameErr := repository.RenameTeam(squad, newName); renameErr != nil {
		http.Error(w, "Не удалось переименовать отряд", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionRenameSquad,
		fmt.Sprintf("отряд %s: «%s» → «%s»", squad.Code, oldName, newName))
	http.Redirect(w, r, "/panel", http.StatusFound)
}

func UnbindSquadChat(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, squadErr := repository.FindOneTeamByCodeAndFraction(chi.URLParam(r, "code"), permissions.FractionId())
	if (squadErr != nil) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if (!permissions.CanRenameSquad(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	if unbindErr := repository.SetTeamChat(squad, nil, nil); (unbindErr != nil) {
		http.Error(w, "Не удалось отвязать чат", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionBindSquadChat,
		fmt.Sprintf("отряд %s: чат отвязан", squad.Code))
	http.Redirect(w, r, "/panel/squads/"+squad.Code, http.StatusFound)
}

// AddSquadMemberOverride pulls a player (by vk id) into this squad
// regardless of what their own profile reports, as long as they already
// have a profile on record in the same fraction — otherwise there's
// nothing to validate the move against.
func AddSquadMemberOverride(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, squadErr := repository.FindOneTeamByCodeAndFraction(chi.URLParam(r, "code"), permissions.FractionId())
	if (squadErr != nil) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if (!permissions.CanManageMembers(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	vkId, vkErr := strconv.ParseUint(strings.TrimSpace(r.FormValue("vk_id")), 10, 64)
	if (vkErr != nil) {
		http.Error(w, "Некорректный VK id", http.StatusBadRequest)
		return
	}

	user := repository.FindOneUserByVkId(int(vkId))
	if (user == nil) {
		http.Error(w, "Игрок не найден — у него должен быть хотя бы один присланный профиль", http.StatusNotFound)
		return
	}

	profile, profileErr := repository.FindLastProfileByVkId(uint(vkId))
	if (profileErr != nil) {
		http.Error(w, "У игрока нет профиля", http.StatusNotFound)
		return
	}

	if (profile.FractionId != squad.FractionId) {
		http.Error(w, "Игрок принадлежит другой фракции", http.StatusBadRequest)
		return
	}

	if setErr := repository.SetUserSquadOverride(user, &squad.Id); (setErr != nil) {
		http.Error(w, "Не удалось переопределить отряд", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionSetSquadOverride,
		fmt.Sprintf("vk%d → отряд %s", vkId, squad.Code))
	http.Redirect(w, r, "/panel/squads/"+squad.Code, http.StatusFound)
}

// RemoveSquadMemberOverride clears a player's override, restoring their
// profile-derived squad. Callable from either side of the move — the
// squad currently holding them via override, or their natural squad —
// but not from an unrelated third squad.
func RemoveSquadMemberOverride(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, squadErr := repository.FindOneTeamByCodeAndFraction(chi.URLParam(r, "code"), permissions.FractionId())
	if (squadErr != nil) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if (!permissions.CanManageMembers(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	vkId, vkErr := strconv.ParseUint(chi.URLParam(r, "vk_id"), 10, 64)
	if (vkErr != nil) {
		http.Error(w, "Некорректный VK id", http.StatusBadRequest)
		return
	}

	user := repository.FindOneUserByVkId(int(vkId))
	if (user == nil) {
		http.Redirect(w, r, "/panel/squads/"+squad.Code, http.StatusFound)
		return
	}

	profile, profileErr := repository.FindLastProfileByVkId(uint(vkId))
	isOverrideTarget := user.SquadOverrideId != nil && *user.SquadOverrideId == squad.Id
	isNaturalSquad := profileErr == nil && profile.TeamId != nil && *profile.TeamId == squad.Id
	if (!isOverrideTarget && !isNaturalSquad) {
		http.Error(w, "Этот игрок не относится к данному отряду", http.StatusBadRequest)
		return
	}

	if setErr := repository.SetUserSquadOverride(user, nil); (setErr != nil) {
		http.Error(w, "Не удалось снять переопределение", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionClearSquadOverride,
		fmt.Sprintf("vk%d: снято переопределение (отряд %s)", vkId, squad.Code))
	http.Redirect(w, r, "/panel/squads/"+squad.Code, http.StatusFound)
}

func DeleteSquad(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanManageSquads()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	squad, squadErr := repository.FindOneTeamByCodeAndFraction(chi.URLParam(r, "code"), permissions.FractionId())
	if (squadErr != nil) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if deleteErr := repository.DeleteTeam(squad.Id); deleteErr != nil {
		http.Error(w, "Не удалось удалить отряд", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionDeleteSquad,
		fmt.Sprintf("удалён отряд %s «%s»", squad.Code, squad.Name))
	http.Redirect(w, r, "/panel", http.StatusFound)
}
