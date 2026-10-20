package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"jiqiren/bot/audit"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/go-chi/chi/v5"
)

type managerView struct {
	Id uint
	VkId uint
	Name string
	AccountType string
	SquadLabel string
}

type managerListData struct {
	Managers []managerView
	Squads []squadOption
}

type squadOption struct {
	Id uint
	Label string
	Selected bool
}

type managerFormData struct {
	Action string
	VkId uint
	AccountType string
	Squads []squadOption
}

func ManagerList(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanManageManagers()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	managers, managersErr := repository.FindManagersByFractionId(permissions.FractionId())
	if (managersErr != nil) {
		http.Error(w, "Не удалось загрузить менеджеров", http.StatusInternalServerError)
		return
	}

	squads, _ := repository.FindTeamsByFractionId(permissions.FractionId())

	vkIds := make([]uint, 0, len(managers))
	for _, manager := range managers {
		vkIds = append(vkIds, manager.VkId)
	}
	names := fetchVkNames(vkIds)

	views := make([]managerView, 0, len(managers))
	for _, manager := range managers {
		accountType := "—"
		if (manager.AccountType != nil) {
			accountType = string(*manager.AccountType)
		}

		views = append(views, managerView{
			Id: manager.Id,
			VkId: manager.VkId,
			Name: names[manager.VkId],
			AccountType: accountType,
			SquadLabel: squadLabelById(squads, manager.SquadId),
		})
	}

	templates.Render(w, "manager_list", managerListData{
		Managers: views,
		Squads: buildSquadOptions(squads, nil),
	})
}

func CreateManager(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanManageManagers()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	vkId, parseErr := strconv.ParseUint(r.FormValue("vk_id"), 10, 64)
	if (parseErr != nil) {
		http.Error(w, "Некорректный VK id", http.StatusBadRequest)
		return
	}

	manager := model.Manager{
		VkId: uint(vkId),
		FractionId: permissions.FractionId(),
	}

	accountType, accountTypeErr := parseAccountType(r.FormValue("account_type"))
	if (accountTypeErr != nil) {
		http.Error(w, "Некорректная роль", http.StatusBadRequest)
		return
	}
	manager.AccountType = accountType

	squadId, squadErr := parseOwnSquadId(r.FormValue("squad_id"), permissions.FractionId())
	if (squadErr != nil) {
		http.Error(w, "Некорректный отряд", http.StatusBadRequest)
		return
	}
	manager.SquadId = squadId

	if createErr := repository.CreateManager(&manager); createErr != nil {
		http.Error(w, "Не удалось создать менеджера", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionCreateManager,
		fmt.Sprintf("добавлен менеджер vk%d (%s)", manager.VkId, roleLabel(manager.AccountType)))
	http.Redirect(w, r, "/panel/managers", http.StatusFound)
}

func EditManagerForm(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanManageManagers()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	manager, managerErr := findOwnManager(r, permissions)
	if (managerErr != nil) {
		http.Error(w, "Менеджер не найден", http.StatusNotFound)
		return
	}

	squads, _ := repository.FindTeamsByFractionId(permissions.FractionId())

	accountType := ""
	if (manager.AccountType != nil) {
		accountType = string(*manager.AccountType)
	}

	templates.Render(w, "manager_form", managerFormData{
		Action: "/panel/managers/" + strconv.FormatUint(uint64(manager.Id), 10) + "/edit",
		VkId: manager.VkId,
		AccountType: accountType,
		Squads: buildSquadOptions(squads, manager.SquadId),
	})
}

func EditManager(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanManageManagers()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	manager, managerErr := findOwnManager(r, permissions)
	if (managerErr != nil) {
		http.Error(w, "Менеджер не найден", http.StatusNotFound)
		return
	}

	vkId, vkErr := strconv.ParseUint(r.FormValue("vk_id"), 10, 64)
	if (vkErr != nil) {
		http.Error(w, "Некорректный VK id", http.StatusBadRequest)
		return
	}
	manager.VkId = uint(vkId)

	accountType, accountTypeErr := parseAccountType(r.FormValue("account_type"))
	if (accountTypeErr != nil) {
		http.Error(w, "Некорректная роль", http.StatusBadRequest)
		return
	}
	manager.AccountType = accountType

	squadId, squadErr := parseOwnSquadId(r.FormValue("squad_id"), permissions.FractionId())
	if (squadErr != nil) {
		http.Error(w, "Некорректный отряд", http.StatusBadRequest)
		return
	}
	manager.SquadId = squadId

	if updateErr := repository.UpdateManager(manager); updateErr != nil {
		http.Error(w, "Не удалось обновить менеджера", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionUpdateManager,
		fmt.Sprintf("обновлён менеджер vk%d (%s)", manager.VkId, roleLabel(manager.AccountType)))
	http.Redirect(w, r, "/panel/managers", http.StatusFound)
}

func DeleteManager(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanManageManagers()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	manager, managerErr := findOwnManager(r, permissions)
	if (managerErr != nil) {
		http.Error(w, "Менеджер не найден", http.StatusNotFound)
		return
	}

	if deleteErr := repository.DeleteManager(manager.Id); deleteErr != nil {
		http.Error(w, "Не удалось удалить менеджера", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionDeleteManager,
		fmt.Sprintf("удалён менеджер vk%d", manager.VkId))
	http.Redirect(w, r, "/panel/managers", http.StatusFound)
}

func findOwnManager(r *http.Request, permissions *auth.Permissions) (*model.Manager, error) {
	id, parseErr := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if (parseErr != nil) {
		return nil, parseErr
	}

	manager, managerErr := repository.FindOneManagerById(uint(id))
	if (managerErr != nil || manager.FractionId != permissions.FractionId()) {
		return nil, managerErr
	}

	return manager, nil
}

func roleLabel(accountType *model.AccountType) string {
	if (accountType == nil) {
		return "—"
	}

	return string(*accountType)
}

func parseAccountType(raw string) (*model.AccountType, error) {
	if (raw == "") {
		return nil, nil
	}

	accountType := model.AccountType(raw)
	switch (accountType) {
	case model.AccountTypeTeamlead, model.AccountTypeDeputy, model.AccountTypeSquadLead:
		return &accountType, nil
	default:
		return nil, strconv.ErrSyntax
	}
}

// parseOwnSquadId resolves an optional squad id from a form value, rejecting
// anything that isn't a squad of the caller's own fraction.
func parseOwnSquadId(raw string, fractionId uint) (*uint, error) {
	if (raw == "") {
		return nil, nil
	}

	squadId, parseErr := strconv.ParseUint(raw, 10, 64)
	if (parseErr != nil) {
		return nil, parseErr
	}

	squad, squadErr := repository.FindOneTeamById(uint(squadId))
	if (squadErr != nil || squad.FractionId != fractionId) {
		return nil, strconv.ErrRange
	}

	squadIdValue := uint(squadId)
	return &squadIdValue, nil
}

func buildSquadOptions(squads []model.Team, selectedSquadId *uint) []squadOption {
	options := make([]squadOption, 0, len(squads))
	for _, squad := range squads {
		options = append(options, squadOption{
			Id: squad.Id,
			Label: squad.Code + " " + squad.Name,
			Selected: selectedSquadId != nil && *selectedSquadId == squad.Id,
		})
	}

	return options
}

// fetchVkNames looks up current first/last names for a batch of VK ids.
// Never cached — called fresh on every manager list load. Failures (rate
// limit, network) degrade gracefully to an empty name per id rather than
// failing the whole page.
func fetchVkNames(vkIds []uint) map[uint]string {
	names := make(map[uint]string, len(vkIds))
	if (len(vkIds) == 0 || vk == nil) {
		return names
	}

	ids := make([]string, len(vkIds))
	for i, vkId := range vkIds {
		ids[i] = strconv.FormatUint(uint64(vkId), 10)
	}

	response, err := vk.UsersGet(api.Params{"user_ids": strings.Join(ids, ",")})
	if (err != nil) {
		return names
	}

	for _, user := range response {
		names[uint(user.ID)] = strings.TrimSpace(user.FirstName + " " + user.LastName)
	}

	return names
}

func squadLabelById(squads []model.Team, id *uint) string {
	if (id == nil) {
		return "—"
	}

	for _, squad := range squads {
		if (squad.Id == *id) {
			return squad.Code
		}
	}

	return "—"
}
