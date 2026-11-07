package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"jiqiren/bot/audit"
	"jiqiren/bot/battletime"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/goalplan"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"

	"github.com/go-chi/chi/v5"
)

type plannedTargetRowView struct {
	Id         uint
	TargetIcon string
	TargetLabel string
	Properties []goalPropertyView
	Time       string
	DateInfo   string
	Applied    bool
	EditHref   string
	DeleteHref string
	HasError   bool
	ErrorText  string
}

type plannedTargetListData struct {
	Title     string
	AddHref   string
	BackHref  string
	CanManage bool
	Goals     []plannedTargetRowView
}

type plannedTargetFormData struct {
	Action        string
	IsEdit        bool
	IsSquad       bool
	TargetOptions []targetIconOption
	TypeOnce      bool
	TypeRecurring bool
	StartDate     string
	MinStartDate  string
	EndDate       string
	Time          string
	Reserve       bool
	Message       string
	HasError      bool
	ErrorText     string
	DeleteHref    string
	BackHref      string
}

const dateOnlyFormat = "2006-01-02"

// ---- squad-scope wrappers ----

func SquadPlannedTargetList(w http.ResponseWriter, r *http.Request) {
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

	basePath := "/panel/squads/" + squad.Code + "/goals"
	renderPlannedTargetList(w, r, squad.FractionId, &squad.Id, "план целей · "+squad.Code+" "+squad.Name,
		basePath, "/panel/squads/"+squad.Code, permissions.CanSetSquadTarget(squad.Id))
}

func NewSquadPlannedTargetForm(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, squadErr := repository.FindOneTeamByCodeAndFraction(chi.URLParam(r, "code"), permissions.FractionId())
	if (squadErr != nil) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if (!permissions.CanSetSquadTarget(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	basePath := "/panel/squads/" + squad.Code + "/goals"
	renderNewPlannedTargetForm(w, r, squad.FractionId, basePath, true, basePath)
}

func CreateSquadPlannedTarget(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, squadErr := repository.FindOneTeamByCodeAndFraction(chi.URLParam(r, "code"), permissions.FractionId())
	if (squadErr != nil) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if (!permissions.CanSetSquadTarget(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	basePath := "/panel/squads/" + squad.Code + "/goals"
	createPlannedTarget(w, r, permissions, squad.FractionId, &squad.Id, true, basePath,
		fmt.Sprintf("отряд %s: добавлена цель в план", squad.Code))
}

func EditSquadPlannedTargetForm(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, target, resolveErr := findOwnSquadPlannedTarget(r, permissions)
	if (resolveErr != nil) {
		http.Error(w, resolveErr.Error(), http.StatusNotFound)
		return
	}

	if (!permissions.CanSetSquadTarget(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	basePath := "/panel/squads/" + squad.Code + "/goals"
	renderEditPlannedTargetForm(w, r, target, basePath, true, basePath)
}

func EditSquadPlannedTarget(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, target, resolveErr := findOwnSquadPlannedTarget(r, permissions)
	if (resolveErr != nil) {
		http.Error(w, resolveErr.Error(), http.StatusNotFound)
		return
	}

	if (!permissions.CanSetSquadTarget(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	basePath := "/panel/squads/" + squad.Code + "/goals"
	updatePlannedTarget(w, r, permissions, target, true, basePath,
		fmt.Sprintf("отряд %s: обновлена цель в плане", squad.Code))
}

func DeleteSquadPlannedTarget(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, target, resolveErr := findOwnSquadPlannedTarget(r, permissions)
	if (resolveErr != nil) {
		http.Error(w, resolveErr.Error(), http.StatusNotFound)
		return
	}

	if (!permissions.CanSetSquadTarget(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	basePath := "/panel/squads/" + squad.Code + "/goals"
	deletePlannedTarget(w, r, permissions, target, basePath,
		fmt.Sprintf("отряд %s: удалена цель из плана", squad.Code))
}

func findOwnSquadPlannedTarget(r *http.Request, permissions *auth.Permissions) (*model.Team, *model.PlannedTarget, error) {
	squad, squadErr := repository.FindOneTeamByCodeAndFraction(chi.URLParam(r, "code"), permissions.FractionId())
	if (squadErr != nil) {
		return nil, nil, fmt.Errorf("Отряд не найден")
	}

	id, idErr := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if (idErr != nil) {
		return nil, nil, fmt.Errorf("Некорректный id цели")
	}

	target, targetErr := repository.FindOnePlannedTargetById(uint(id))
	if (targetErr != nil || target.SquadId == nil || *target.SquadId != squad.Id) {
		return nil, nil, fmt.Errorf("Цель не найдена")
	}

	return squad, target, nil
}

// ---- fraction-scope wrappers ----

func FractionPlannedTargetList(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanViewFraction()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	renderPlannedTargetList(w, r, permissions.FractionId(), nil, "план целей команды",
		"/panel/fraction/goals", "/panel/fraction", permissions.CanSetFractionTarget())
}

func NewFractionPlannedTargetForm(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanSetFractionTarget()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	basePath := "/panel/fraction/goals"
	renderNewPlannedTargetForm(w, r, permissions.FractionId(), basePath, false, basePath)
}

func CreateFractionPlannedTarget(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanSetFractionTarget()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	basePath := "/panel/fraction/goals"
	createPlannedTarget(w, r, permissions, permissions.FractionId(), nil, false, basePath,
		"фракция: добавлена цель в план")
}

func EditFractionPlannedTargetForm(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanSetFractionTarget()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	target, resolveErr := findOwnFractionPlannedTarget(r, permissions)
	if (resolveErr != nil) {
		http.Error(w, resolveErr.Error(), http.StatusNotFound)
		return
	}

	basePath := "/panel/fraction/goals"
	renderEditPlannedTargetForm(w, r, target, basePath, false, basePath)
}

func EditFractionPlannedTarget(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanSetFractionTarget()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	target, resolveErr := findOwnFractionPlannedTarget(r, permissions)
	if (resolveErr != nil) {
		http.Error(w, resolveErr.Error(), http.StatusNotFound)
		return
	}

	basePath := "/panel/fraction/goals"
	updatePlannedTarget(w, r, permissions, target, false, basePath, "фракция: обновлена цель в плане")
}

func DeleteFractionPlannedTarget(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanSetFractionTarget()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	target, resolveErr := findOwnFractionPlannedTarget(r, permissions)
	if (resolveErr != nil) {
		http.Error(w, resolveErr.Error(), http.StatusNotFound)
		return
	}

	basePath := "/panel/fraction/goals"
	deletePlannedTarget(w, r, permissions, target, basePath, "фракция: удалена цель из плана")
}

func findOwnFractionPlannedTarget(r *http.Request, permissions *auth.Permissions) (*model.PlannedTarget, error) {
	id, idErr := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if (idErr != nil) {
		return nil, fmt.Errorf("Некорректный id цели")
	}

	target, targetErr := repository.FindOnePlannedTargetById(uint(id))
	if (targetErr != nil || target.SquadId != nil || target.FractionId != permissions.FractionId()) {
		return nil, fmt.Errorf("Цель не найдена")
	}

	return target, nil
}

// ---- shared core ----

func renderPlannedTargetList(w http.ResponseWriter, r *http.Request, fractionId uint, squadId *uint, title string, basePath string, backPath string, canManage bool) {
	var targets []model.PlannedTarget
	var err error
	if (squadId != nil) {
		targets, err = repository.FindPlannedTargetsBySquadId(*squadId)
	} else {
		targets, err = repository.FindPlannedTargetsByFractionId(fractionId)
	}
	if (err != nil) {
		http.Error(w, "Не удалось загрузить план целей", http.StatusInternalServerError)
		return
	}

	allFractions, fractionsErr := repository.FindAllFractions()
	if (fractionsErr != nil) {
		http.Error(w, "Ошибка загрузки фракций", http.StatusInternalServerError)
		return
	}

	rows := make([]plannedTargetRowView, 0, len(targets))
	for _, target := range targets {
		rows = append(rows, buildPlannedTargetRowView(target, allFractions, basePath))
	}

	templates.Render(w, r, "goal_plan_list", plannedTargetListData{
		Title:     title,
		AddHref:   basePath + "/new",
		BackHref:  backPath,
		CanManage: canManage,
		Goals:     rows,
	})
}

func buildPlannedTargetRowView(target model.PlannedTarget, allFractions []model.Fraction, basePath string) plannedTargetRowView {
	targetValue := target.Target
	moscow := battletime.Moscow

	var dateInfo string
	if (target.Type == model.PlannedTargetTypeOnce) {
		dateInfo = target.StartDate.In(moscow).Format("02.01.2006")
	} else {
		dateInfo = "с " + target.StartDate.In(moscow).Format("02.01.2006")
		if (target.EndDate != nil) {
			dateInfo += " по " + target.EndDate.In(moscow).Format("02.01.2006")
		}
	}

	id := strconv.FormatUint(uint64(target.Id), 10)

	return plannedTargetRowView{
		Id:          target.Id,
		TargetIcon:  targetIcon(allFractions, &targetValue),
		TargetLabel: targetLabel(allFractions, &targetValue),
		Properties:  goalProperties(target),
		Time:        fmt.Sprintf("%02d:%02d", target.Hour, target.Minute),
		DateInfo:    dateInfo,
		Applied:     target.Type == model.PlannedTargetTypeOnce && target.LastAppliedDate != nil,
		EditHref:    basePath + "/" + id + "/edit",
		DeleteHref:  basePath + "/" + id + "/delete",
		HasError:    target.LastError != "",
		ErrorText:   target.LastError,
	}
}

// goalPropertyView.Kind names which inline SVG the "goal_icon" template
// partial should draw ("repeat" / "message" / "reserve") — properties are
// rendered as site-colored SVG rather than emoji, see goal_plan_list.html.tmpl.
type goalPropertyView struct {
	Kind  string
	Label string
}

// goalProperties lists the active badge-worthy properties of a planned
// target, shared by both the compact card and the full-width row.
func goalProperties(target model.PlannedTarget) []goalPropertyView {
	properties := make([]goalPropertyView, 0, 3)
	if (target.Type == model.PlannedTargetTypeRecurring) {
		properties = append(properties, goalPropertyView{Kind: "repeat", Label: "повторяется"})
	}
	if (target.Message != "") {
		properties = append(properties, goalPropertyView{Kind: "message", Label: "с сообщением"})
	}
	if (target.Reserve) {
		properties = append(properties, goalPropertyView{Kind: "reserve", Label: "резервная"})
	}

	return properties
}

type goalCardView struct {
	Icon       string
	Properties []goalPropertyView
	Time       string
	Date       string
	EditHref   string
	HasError   bool
	ErrorText  string
}

type goalPlanSectionData struct {
	Cards     []goalCardView
	AddHref   string
	MoreHref  string
	MoreLabel string
	HasAny    bool
}

// buildGoalPlanSection is the compact "план целей" block embedded on both
// the squad and fraction pages: up to 3 nearest upcoming cards (reserve
// entries included even though they might not actually fire — see
// goalplan.NextOccurrence), plus a trailing link to the full list worded
// depending on whether anything beyond those 3 exists.
func buildGoalPlanSection(targets []model.PlannedTarget, now time.Time, allFractions []model.Fraction, basePath string, canManage bool) goalPlanSectionData {
	const previewLimit = 3

	upcoming := goalplan.BuildUpcoming(targets, now, previewLimit)
	moscow := battletime.Moscow

	cards := make([]goalCardView, 0, len(upcoming))
	for _, entry := range upcoming {
		targetValue := entry.Target.Target
		id := strconv.FormatUint(uint64(entry.Target.Id), 10)

		date := ""
		if (entry.Moment.Sub(now) > 24*time.Hour) {
			date = entry.Moment.In(moscow).Format("02.01")
		}

		cards = append(cards, goalCardView{
			Icon:       targetIcon(allFractions, &targetValue),
			Properties: goalProperties(entry.Target),
			Time:       entry.Moment.In(moscow).Format("15:04"),
			Date:       date,
			EditHref:   basePath + "/" + id + "/edit",
			HasError:   entry.Target.LastError != "",
			ErrorText:  entry.Target.LastError,
		})
	}

	section := goalPlanSectionData{
		Cards:  cards,
		HasAny: len(targets) > 0,
	}

	if (canManage) {
		section.AddHref = basePath + "/new"
	}

	if (len(targets) > 0) {
		section.MoreHref = basePath
		remainder := len(targets) - len(cards)
		if (remainder > 0) {
			section.MoreLabel = fmt.Sprintf("ещё %d целей", remainder)
		} else {
			section.MoreLabel = "все цели"
		}
	}

	return section
}

func renderNewPlannedTargetForm(w http.ResponseWriter, r *http.Request, fractionId uint, action string, isSquad bool, backPath string) {
	allFractions, err := repository.FindAllFractions()
	if (err != nil) {
		http.Error(w, "Ошибка загрузки фракций", http.StatusInternalServerError)
		return
	}

	today := battletime.Today(time.Now()).Format(dateOnlyFormat)

	templates.Render(w, r, "goal_plan_form", plannedTargetFormData{
		Action:        action,
		IsSquad:       isSquad,
		TargetOptions: buildTargetIconOptions(allFractions, fractionId, model.TargetNone),
		TypeOnce:      true,
		StartDate:     today,
		MinStartDate:  today,
		Time:          "18:00",
		BackHref:      backPath,
	})
}

func renderEditPlannedTargetForm(w http.ResponseWriter, r *http.Request, target *model.PlannedTarget, basePath string, isSquad bool, backPath string) {
	allFractions, err := repository.FindAllFractions()
	if (err != nil) {
		http.Error(w, "Ошибка загрузки фракций", http.StatusInternalServerError)
		return
	}

	moscow := battletime.Moscow
	endDate := ""
	if (target.EndDate != nil) {
		endDate = target.EndDate.In(moscow).Format(dateOnlyFormat)
	}

	id := strconv.FormatUint(uint64(target.Id), 10)

	templates.Render(w, r, "goal_plan_form", plannedTargetFormData{
		Action:        basePath + "/" + id + "/edit",
		IsEdit:        true,
		IsSquad:       isSquad,
		TargetOptions: buildTargetIconOptions(allFractions, target.FractionId, target.Target),
		TypeOnce:      target.Type == model.PlannedTargetTypeOnce,
		TypeRecurring: target.Type == model.PlannedTargetTypeRecurring,
		StartDate:     target.StartDate.In(moscow).Format(dateOnlyFormat),
		EndDate:       endDate,
		Time:          fmt.Sprintf("%02d:%02d", target.Hour, target.Minute),
		Reserve:       target.Reserve,
		Message:       target.Message,
		HasError:      target.LastError != "",
		ErrorText:     target.LastError,
		DeleteHref:    basePath + "/" + id + "/delete",
		BackHref:      backPath,
	})
}

// parsePlannedTargetForm reads and validates the fields shared by create
// and edit — the target, schedule (once vs. recurring, dates, time) and
// the reserve/message flags, the latter gated to squad scope only.
func parsePlannedTargetForm(r *http.Request, isSquad bool) (*model.PlannedTarget, error) {
	targetValue, parseErr := strconv.ParseUint(r.FormValue("target"), 10, 64)
	if (parseErr != nil || !model.IsValidTarget(uint(targetValue))) {
		return nil, fmt.Errorf("Некорректная цель")
	}

	var targetType model.PlannedTargetType
	switch (r.FormValue("type")) {
	case string(model.PlannedTargetTypeOnce):
		targetType = model.PlannedTargetTypeOnce
	case string(model.PlannedTargetTypeRecurring):
		targetType = model.PlannedTargetTypeRecurring
	default:
		return nil, fmt.Errorf("Некорректный тип цели")
	}

	hour, minute, timeErr := parseScheduleTime(r.FormValue("time"))
	if (timeErr != nil) {
		return nil, fmt.Errorf("Некорректное время")
	}

	startDate, startErr := time.Parse(dateOnlyFormat, r.FormValue("start_date"))
	if (startErr != nil) {
		return nil, fmt.Errorf("Некорректная дата")
	}

	var endDate *time.Time
	if (targetType == model.PlannedTargetTypeRecurring) {
		if endRaw := strings.TrimSpace(r.FormValue("end_date")); (endRaw != "") {
			parsed, endErr := time.Parse(dateOnlyFormat, endRaw)
			if (endErr != nil) {
				return nil, fmt.Errorf("Некорректная дата окончания")
			}
			if (parsed.Before(startDate)) {
				return nil, fmt.Errorf("Дата окончания раньше даты начала")
			}
			endDate = &parsed
		}
	}

	message := ""
	if (isSquad) {
		message = strings.TrimSpace(r.FormValue("message"))
	} else if (strings.TrimSpace(r.FormValue("message")) != "") {
		return nil, fmt.Errorf("У цели фракции не может быть сообщения")
	}

	return &model.PlannedTarget{
		Target:    uint(targetValue),
		Type:      targetType,
		StartDate: startDate,
		EndDate:   endDate,
		Hour:      hour,
		Minute:    minute,
		Reserve:   r.FormValue("reserve") == "on",
		Message:   message,
	}, nil
}

// rollPlannedTargetStartDate enforces "no start date in the past" at
// creation time — a start date before today (MSK) is rejected outright; a
// start date of today whose hour:minute (MSK) has already gone by rolls
// forward to tomorrow instead of being accepted as-is, since "today at a
// time that's already passed" isn't a sensible first occurrence to create.
func rollPlannedTargetStartDate(startDate time.Time, hour uint, minute uint, now time.Time) (time.Time, error) {
	today := battletime.Today(now)
	if (startDate.Before(today)) {
		return time.Time{}, fmt.Errorf("Дата начала не может быть в прошлом")
	}

	if (startDate.Equal(today) && battletime.IsPastToday(hour, minute, now)) {
		return startDate.AddDate(0, 0, 1), nil
	}

	return startDate, nil
}

func createPlannedTarget(w http.ResponseWriter, r *http.Request, permissions *auth.Permissions, fractionId uint, squadId *uint, isSquad bool, basePath string, auditDetails string) {
	parsed, parseErr := parsePlannedTargetForm(r, isSquad)
	if (parseErr != nil) {
		http.Error(w, parseErr.Error(), http.StatusBadRequest)
		return
	}

	startDate, startErr := rollPlannedTargetStartDate(parsed.StartDate, parsed.Hour, parsed.Minute, time.Now())
	if (startErr != nil) {
		http.Error(w, startErr.Error(), http.StatusBadRequest)
		return
	}
	parsed.StartDate = startDate

	parsed.FractionId = fractionId
	parsed.SquadId = squadId

	if createErr := repository.CreatePlannedTarget(parsed); (createErr != nil) {
		http.Error(w, "Не удалось добавить цель в план", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionCreatePlannedTarget, auditDetails)
	http.Redirect(w, r, basePath, http.StatusFound)
}

func updatePlannedTarget(w http.ResponseWriter, r *http.Request, permissions *auth.Permissions, target *model.PlannedTarget, isSquad bool, basePath string, auditDetails string) {
	parsed, parseErr := parsePlannedTargetForm(r, isSquad)
	if (parseErr != nil) {
		http.Error(w, parseErr.Error(), http.StatusBadRequest)
		return
	}

	if updateErr := repository.UpdatePlannedTarget(target, *parsed); (updateErr != nil) {
		http.Error(w, "Не удалось обновить цель", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionUpdatePlannedTarget, auditDetails)
	http.Redirect(w, r, basePath, http.StatusFound)
}

func deletePlannedTarget(w http.ResponseWriter, r *http.Request, permissions *auth.Permissions, target *model.PlannedTarget, basePath string, auditDetails string) {
	if deleteErr := repository.DeletePlannedTarget(target.Id); (deleteErr != nil) {
		http.Error(w, "Не удалось удалить цель", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionDeletePlannedTarget, auditDetails)
	http.Redirect(w, r, basePath, http.StatusFound)
}
