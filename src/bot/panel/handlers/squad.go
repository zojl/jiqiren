package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"jiqiren/bot/audit"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"

	"github.com/go-chi/chi/v5"
)

// moscow is a fixed +3 offset rather than a tzdata-backed location: Moscow
// has not observed DST since 2014, and the prod image (golang:alpine) isn't
// guaranteed to ship tzdata. Matches the UTC offset already assumed in
// repository.FindLastProfileByVkIdForBattleDate (battle at 15:00 UTC = 18:00 MSK).
var moscow = time.FixedZone("MSK", 3*60*60)

const battleHourUTC = 15 // 18:00 MSK
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
	ChatId *int64
	TargetLabel string
	CurrentTargetIcon string
	TargetOptions []targetIconOption
	HasToken bool
	CanSetTarget bool
	CanRename bool
	CanSetToken bool
	ShowBackLink bool
	Members []squadMemberView
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

	members, membersErr := buildSquadMembers(squad, allFractions)
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
		ChatId: squad.ChatId,
		TargetLabel: targetLabel(allFractions, squad.Target),
		CurrentTargetIcon: targetIcon(allFractions, squad.Target),
		TargetOptions: buildTargetIconOptions(allFractions, squad.FractionId, target),
		HasToken: squad.TargetToken != "",
		CanSetTarget: permissions.CanSetSquadTarget(squad.Id),
		CanRename: permissions.CanRenameSquad(squad.Id),
		CanSetToken: permissions.CanSetSquadToken(squad.Id),
		ShowBackLink: !isHome || homeSquadId != squad.Id,
		Members: members,
	})
}

// buildSquadMembers derives the squad roster from each player's own most
// recently reported profile — there is no authoritative membership list.
// A player who moved on now shows a different TeamId on their latest
// profile and is excluded; one who simply stopped reporting is kept for a
// month, marked stale after a week.
func buildSquadMembers(squad *model.Team, allFractions []model.Fraction) ([]squadMemberView, error) {
	profiles, err := repository.FindLatestProfilesForTeam(squad.Id)
	if (err != nil) {
		return nil, err
	}

	userIds := make([]uint, 0, len(profiles))
	for _, profile := range profiles {
		userIds = append(userIds, profile.UserId)
	}

	users, _ := repository.FindUsersByIds(userIds)
	vkIdByUserId := make(map[uint]uint, len(users))
	for _, user := range users {
		vkIdByUserId[user.Id] = user.VkId
	}

	now := time.Now()
	nextBattle := nextBattleTime(now)
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
	for _, profile := range profiles {
		if (profile.TeamId == nil || *profile.TeamId != squad.Id) {
			continue
		}

		inactiveFor := now.Sub(profile.MessageDate)
		if (inactiveFor > inactiveHiddenAfter) {
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
		if (profileTargetMatchesSquad(profile, squad)) {
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

	return members, nil
}

// formatReportMoney renders the battle's money delta as signed parts —
// income, intercepted-transaction bonus (++), loss from a breached defense,
// and loss from a failed attack — omitting whichever parts were zero.
func formatReportMoney(report model.Report) string {
	parts := make([]string, 0, 4)
	if (report.RewardMoney > 0) {
		parts = append(parts, fmt.Sprintf("+%d", report.RewardMoney))
	}
	if (report.TransactionMoney > 0) {
		parts = append(parts, fmt.Sprintf("++%d", report.TransactionMoney))
	}
	if (report.LostMoney > 0) {
		parts = append(parts, fmt.Sprintf("-%d(деф)", report.LostMoney))
	}
	if (report.AttackLostMoney > 0) {
		parts = append(parts, fmt.Sprintf("-%d(атака)", report.AttackLostMoney))
	}

	if (len(parts) == 0) {
		return "0"
	}

	return strings.Join(parts, " ")
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

func nextBattleTime(now time.Time) time.Time {
	nowUTC := now.UTC()
	battle := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), battleHourUTC, 0, 0, 0, time.UTC)
	if (!nowUTC.Before(battle)) {
		battle = battle.Add(24 * time.Hour)
	}

	return battle
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

// profileTargetMatchesSquad compares a player's chosen battle target to the
// squad's directive. A player's own fraction as target means "defending"
// (that's how the in-game 🔐 icon is parsed, see getTarget in
// plain/profile.go), matching the squad target sentinel model.TargetDefense.
func profileTargetMatchesSquad(profile model.Profile, squad *model.Team) bool {
	if (profile.TargetId == nil) {
		return false
	}

	if (squad.Target == nil || *squad.Target == model.TargetNone) {
		return true
	}

	if (*squad.Target == model.TargetDefense) {
		return *profile.TargetId == squad.FractionId
	}

	return *profile.TargetId == *squad.Target
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
