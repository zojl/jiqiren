package handlers

import (
	"net/http"
	"strconv"

	"jiqiren/bot/audit"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/external/hackerwars"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"
)

type messageFormData struct {
	Action  string
	Message string
}

func FractionMessageForm(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanSetFractionMessage()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	fraction, err := repository.FindOneFractionById(permissions.FractionId())
	if (err != nil) {
		http.Error(w, "Фракция не найдена", http.StatusInternalServerError)
		return
	}

	templates.Render(w, r, "message_form", messageFormData{
		Action:  "/panel/fraction/message",
		Message: fraction.Message,
	})
}

func SetFractionMessage(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanSetFractionMessage()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	fraction, fractionErr := repository.FindOneFractionById(permissions.FractionId())
	if (fractionErr != nil) {
		http.Error(w, "Фракция не найдена", http.StatusInternalServerError)
		return
	}

	if (fraction.TargetToken == "") {
		http.Error(w, "Для команды не задан токен", http.StatusBadRequest)
		return
	}

	message := r.FormValue("message")
	team := strconv.FormatUint(uint64(fraction.Id), 10)

	if setErr := hackerwars.SetTeamMessage(fraction.TargetToken, team, message); (setErr != nil) {
		http.Error(w, "Внешний API отклонил запрос: "+setErr.Error(), http.StatusBadGateway)
		return
	}

	repository.SetFractionMessage(fraction, message)
	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionSetFractionMessage, "обновлено сообщение команды")
	http.Redirect(w, r, "/panel/fraction", http.StatusFound)
}
