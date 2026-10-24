package handlers

import (
	"net/http"
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
	model.AuditActionBindSquadChat:     "привязка чата отряда",
	model.AuditActionCreateSchedule:    "добавление рассылки",
	model.AuditActionUpdateSchedule:    "обновление рассылки",
	model.AuditActionDeleteSchedule:    "удаление рассылки",
}

type auditLogView struct {
	Time    string
	Who     string
	Action  string
	Details string
}

type auditLogData struct {
	Entries []auditLogView
}

func AuditLog(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanViewAuditLog()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	entries, entriesErr := repository.FindAuditLogsByFractionSince(permissions.FractionId(), time.Now().Add(-auditLogWindow))
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

	templates.Render(w, "audit_log", auditLogData{Entries: views})
}
