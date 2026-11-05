package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"
)

const auditLogWindow = 14 * 24 * time.Hour

var auditActionLabels = map[model.AuditAction]string{
	model.AuditActionLoginRequest:      "запрос входа",
	model.AuditActionLogin:             "вход",
	model.AuditActionSetTarget:         "установка цели",
	model.AuditActionSetSquadToken:     "токен отряда",
	model.AuditActionSetFractionToken:  "токен команды",
	model.AuditActionRenameSquad:       "обновление отряда",
	model.AuditActionCreateSquad:       "добавление отряда",
	model.AuditActionDeleteSquad:       "удаление отряда",
	model.AuditActionCreateManager:     "добавление менеджера",
	model.AuditActionUpdateManager:     "обновление менеджера",
	model.AuditActionDeleteManager:     "удаление менеджера",
	model.AuditActionViewBalance:       "просмотр баланса",
	model.AuditActionViewActualBudget:  "просмотр текущего бюджета",
	model.AuditActionViewBudgetChart:   "просмотр графика бюджета",
	model.AuditActionViewSquad:         "просмотр отряда",
	model.AuditActionBindSquadChat:     "привязка чата отряда",
	model.AuditActionCreateSchedule:    "добавление рассылки",
	model.AuditActionUpdateSchedule:    "обновление рассылки",
	model.AuditActionDeleteSchedule:    "удаление рассылки",
	model.AuditActionSetSquadOverride:   "переопределение отряда",
	model.AuditActionClearSquadOverride: "снятие переопределения отряда",
}

type auditLogView struct {
	Time    string
	Who     string
	Action  string
	Details string
}

type auditFilterOption struct {
	Value    string
	Label    string
	Selected bool
}

type auditLogData struct {
	Entries        []auditLogView
	ManagerOptions []auditFilterOption
	ActionOptions  []auditFilterOption
}

func AuditLog(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanViewAuditLog()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	selectedManagerId := r.URL.Query().Get("manager_id")
	selectedAction := r.URL.Query().Get("action")

	filter := repository.AuditLogFilter{}
	if (selectedManagerId != "") {
		if id, err := strconv.ParseUint(selectedManagerId, 10, 64); (err == nil) {
			managerId := uint(id)
			filter.ManagerId = &managerId
		}
	}

	if (selectedAction != "") {
		action := model.AuditAction(selectedAction)
		filter.Action = &action
	}

	entries, entriesErr := repository.FindAuditLogs(permissions.FractionId(), time.Now().Add(-auditLogWindow), filter)
	if (entriesErr != nil) {
		http.Error(w, "Не удалось загрузить журнал", http.StatusInternalServerError)
		return
	}

	vkIds := make([]uint, 0, len(entries))
	for _, entry := range entries {
		vkIds = append(vkIds, entry.VkId)
	}
	names := fetchVkNames(vkIds)

	views := make([]auditLogView, 0, len(entries))
	for _, entry := range entries {
		who := strconv.FormatUint(uint64(entry.VkId), 10)
		if (names[entry.VkId] != "") {
			who = names[entry.VkId] + " (vk" + who + ")"
		}

		action, known := auditActionLabels[entry.Action]
		if (!known) {
			action = string(entry.Action)
		}

		views = append(views, auditLogView{
			Time: entry.CreatedAt.Local().Format("02.01.2006 15:04:05"),
			Who: who,
			Action: action,
			Details: entry.Details,
		})
	}

	templates.Render(w, r, "audit_log", auditLogData{
		Entries:        views,
		ManagerOptions: managerFilterOptions(permissions.FractionId(), selectedManagerId),
		ActionOptions:  actionFilterOptions(selectedAction),
	})
}

func managerFilterOptions(fractionId uint, selectedManagerId string) []auditFilterOption {
	managers, err := repository.FindManagersByFractionId(fractionId)
	if (err != nil) {
		return nil
	}

	vkIds := make([]uint, 0, len(managers))
	for _, manager := range managers {
		vkIds = append(vkIds, manager.VkId)
	}
	names := fetchVkNames(vkIds)

	options := make([]auditFilterOption, 0, len(managers))
	for _, manager := range managers {
		idStr := strconv.FormatUint(uint64(manager.Id), 10)
		label := names[manager.VkId]
		if (label == "") {
			label = "vk" + strconv.FormatUint(uint64(manager.VkId), 10)
		}

		options = append(options, auditFilterOption{Value: idStr, Label: label, Selected: idStr == selectedManagerId})
	}

	sort.Slice(options, func(i, j int) bool { return options[i].Label < options[j].Label })

	return options
}

func actionFilterOptions(selectedAction string) []auditFilterOption {
	options := make([]auditFilterOption, 0, len(auditActionLabels))
	for action, label := range auditActionLabels {
		value := string(action)
		options = append(options, auditFilterOption{Value: value, Label: label, Selected: value == selectedAction})
	}

	sort.Slice(options, func(i, j int) bool { return options[i].Label < options[j].Label })

	return options
}
