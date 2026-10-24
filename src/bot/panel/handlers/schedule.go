package handlers

import (
	"fmt"
	"net/http"
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

var scheduleTypeLabels = map[model.ScheduledMessageType]string{
	model.ScheduledMessageTypeBasic: "обычное",
	model.ScheduledMessageTypeMissingProfiles: "недостающие профили",
	model.ScheduledMessageTypeMissingReports: "недостающие отчёты",
	model.ScheduledMessageTypeSleepReminder: "напоминание о сне",
}

type scheduleTypeOption struct {
	Value string
	Label string
	Selected bool
}

type scheduledMessageView struct {
	Id uint
	Time string
	TypeLabel string
	Text string
}

type scheduleListData struct {
	SquadCode string
	SquadName string
	Messages []scheduledMessageView
}

type scheduleFormData struct {
	Action string
	SquadCode string
	Time string
	Text string
	Types []scheduleTypeOption
}

func buildScheduleTypeOptions(selected model.ScheduledMessageType) []scheduleTypeOption {
	types := []model.ScheduledMessageType{
		model.ScheduledMessageTypeBasic,
		model.ScheduledMessageTypeMissingProfiles,
		model.ScheduledMessageTypeMissingReports,
		model.ScheduledMessageTypeSleepReminder,
	}

	options := make([]scheduleTypeOption, 0, len(types))
	for _, messageType := range types {
		options = append(options, scheduleTypeOption{
			Value: string(messageType),
			Label: scheduleTypeLabels[messageType],
			Selected: messageType == selected,
		})
	}

	return options
}

func parseScheduleType(raw string) (model.ScheduledMessageType, error) {
	messageType := model.ScheduledMessageType(raw)
	if _, known := scheduleTypeLabels[messageType]; (!known) {
		return "", fmt.Errorf("unknown schedule type: %s", raw)
	}

	return messageType, nil
}

func SquadSchedule(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, squadErr := repository.FindOneTeamByCodeAndFraction(chi.URLParam(r, "code"), permissions.FractionId())
	if (squadErr != nil) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if (!permissions.CanManageSchedule(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	messages, messagesErr := repository.FindScheduledMessagesBySquadId(squad.Id)
	if (messagesErr != nil) {
		http.Error(w, "Не удалось загрузить расписание", http.StatusInternalServerError)
		return
	}

	views := make([]scheduledMessageView, 0, len(messages))
	for _, message := range messages {
		views = append(views, scheduledMessageView{
			Id: message.Id,
			Time: fmt.Sprintf("%02d:%02d", message.SendHour, message.SendMinute),
			TypeLabel: scheduleTypeLabels[message.Type],
			Text: message.Text,
		})
	}

	templates.Render(w, "squad_schedule", scheduleListData{
		SquadCode: squad.Code,
		SquadName: squad.Name,
		Messages: views,
	})
}

func NewScheduledMessageForm(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, squadErr := repository.FindOneTeamByCodeAndFraction(chi.URLParam(r, "code"), permissions.FractionId())
	if (squadErr != nil) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if (!permissions.CanManageSchedule(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	templates.Render(w, "schedule_form", scheduleFormData{
		Action: "/panel/squads/" + squad.Code + "/schedule",
		SquadCode: squad.Code,
		Types: buildScheduleTypeOptions(model.ScheduledMessageTypeBasic),
	})
}

func CreateScheduledMessage(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, squadErr := repository.FindOneTeamByCodeAndFraction(chi.URLParam(r, "code"), permissions.FractionId())
	if (squadErr != nil) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if (!permissions.CanManageSchedule(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	messageType, typeErr := parseScheduleType(r.FormValue("type"))
	if (typeErr != nil) {
		http.Error(w, "Некорректный тип рассылки", http.StatusBadRequest)
		return
	}

	hour, minute, timeErr := parseScheduleTime(r.FormValue("time"))
	if (timeErr != nil) {
		http.Error(w, "Некорректное время", http.StatusBadRequest)
		return
	}

	text := strings.TrimSpace(r.FormValue("text"))
	if (text == "" && messageType == model.ScheduledMessageTypeBasic) {
		http.Error(w, "Текст не может быть пустым", http.StatusBadRequest)
		return
	}

	message := model.ScheduledMessage{
		SquadId: squad.Id,
		Type: messageType,
		SendHour: hour,
		SendMinute: minute,
		Text: text,
	}

	if createErr := repository.CreateScheduledMessage(&message); (createErr != nil) {
		http.Error(w, "Не удалось создать рассылку", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionCreateSchedule,
		fmt.Sprintf("отряд %s: добавлена рассылка на %02d:%02d", squad.Code, hour, minute))
	http.Redirect(w, r, "/panel/squads/"+squad.Code+"/schedule", http.StatusFound)
}

func EditScheduledMessageForm(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, message, resolveErr := findOwnScheduledMessage(r, permissions)
	if (resolveErr != nil) {
		http.Error(w, resolveErr.Error(), http.StatusNotFound)
		return
	}

	if (!permissions.CanManageSchedule(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	templates.Render(w, "schedule_form", scheduleFormData{
		Action: "/panel/squads/" + squad.Code + "/schedule/" + strconv.FormatUint(uint64(message.Id), 10) + "/edit",
		SquadCode: squad.Code,
		Time: fmt.Sprintf("%02d:%02d", message.SendHour, message.SendMinute),
		Text: message.Text,
		Types: buildScheduleTypeOptions(message.Type),
	})
}

func EditScheduledMessage(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, message, resolveErr := findOwnScheduledMessage(r, permissions)
	if (resolveErr != nil) {
		http.Error(w, resolveErr.Error(), http.StatusNotFound)
		return
	}

	if (!permissions.CanManageSchedule(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	messageType, typeErr := parseScheduleType(r.FormValue("type"))
	if (typeErr != nil) {
		http.Error(w, "Некорректный тип рассылки", http.StatusBadRequest)
		return
	}

	hour, minute, timeErr := parseScheduleTime(r.FormValue("time"))
	if (timeErr != nil) {
		http.Error(w, "Некорректное время", http.StatusBadRequest)
		return
	}

	text := strings.TrimSpace(r.FormValue("text"))
	if (text == "" && messageType == model.ScheduledMessageTypeBasic) {
		http.Error(w, "Текст не может быть пустым", http.StatusBadRequest)
		return
	}

	if updateErr := repository.UpdateScheduledMessage(message, messageType, hour, minute, text); (updateErr != nil) {
		http.Error(w, "Не удалось обновить рассылку", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionUpdateSchedule,
		fmt.Sprintf("отряд %s: обновлена рассылка на %02d:%02d", squad.Code, hour, minute))
	http.Redirect(w, r, "/panel/squads/"+squad.Code+"/schedule", http.StatusFound)
}

func DeleteScheduledMessage(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, message, resolveErr := findOwnScheduledMessage(r, permissions)
	if (resolveErr != nil) {
		http.Error(w, resolveErr.Error(), http.StatusNotFound)
		return
	}

	if (!permissions.CanManageSchedule(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	if deleteErr := repository.DeleteScheduledMessage(message.Id); (deleteErr != nil) {
		http.Error(w, "Не удалось удалить рассылку", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionDeleteSchedule,
		fmt.Sprintf("отряд %s: удалена рассылка на %02d:%02d", squad.Code, message.SendHour, message.SendMinute))
	http.Redirect(w, r, "/panel/squads/"+squad.Code+"/schedule", http.StatusFound)
}

func findOwnScheduledMessage(r *http.Request, permissions *auth.Permissions) (*model.Team, *model.ScheduledMessage, error) {
	squad, squadErr := repository.FindOneTeamByCodeAndFraction(chi.URLParam(r, "code"), permissions.FractionId())
	if (squadErr != nil) {
		return nil, nil, fmt.Errorf("Отряд не найден")
	}

	id, idErr := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if (idErr != nil) {
		return nil, nil, fmt.Errorf("Некорректный id рассылки")
	}

	message, messageErr := repository.FindOneScheduledMessageById(uint(id))
	if (messageErr != nil || message.SquadId != squad.Id) {
		return nil, nil, fmt.Errorf("Рассылка не найдена")
	}

	return squad, message, nil
}

func parseScheduleTime(value string) (uint, uint, error) {
	parsed, err := time.Parse("15:04", value)
	if (err != nil) {
		return 0, 0, err
	}

	return uint(parsed.Hour()), uint(parsed.Minute()), nil
}
