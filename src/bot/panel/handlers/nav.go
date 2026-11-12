package handlers

import (
	"net/http"
	"strings"

	"jiqiren/bot/database/repository"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"
)

func init() {
	templates.BuildNav = buildNav
}

// buildNav assembles the permission-filtered top navigation for the
// current request. r.URL.Path has already had the /panel mount prefix
// stripped (see bot.go's http.StripPrefix), so it's compared against the
// same relative paths the router registers, while Href carries the full
// /panel-prefixed path the browser needs.
func buildNav(r *http.Request) []templates.NavItem {
	permissions := auth.PermissionsFromRequest(r)
	path := r.URL.Path

	items := make([]templates.NavItem, 0, 8)

	if (permissions.CanViewDashboard()) {
		items = append(items, templates.NavItem{Label: "Отряды", Href: "/panel/", Active: path == "/"})
	}

	if (permissions.CanViewBalance()) {
		items = append(items, templates.NavItem{Label: "Бюджет", Href: "/panel/fraction/budget", Active: strings.HasPrefix(path, "/fraction/budget")})
	}

	if (permissions.CanViewFraction()) {
		items = append(items, templates.NavItem{Label: "Команда", Href: "/panel/fraction", Active: path == "/fraction"})
	}

	if squadId, hasOwnSquad := permissions.OwnSquadId(); (hasOwnSquad) {
		if squad, err := repository.FindOneTeamById(squadId); (err == nil) {
			href := "/squads/" + squad.Code
			items = append(items, templates.NavItem{Label: "Мой отряд", Href: "/panel" + href, Active: path == href})
		}
	}

	if (permissions.CanManageManagers()) {
		items = append(items, templates.NavItem{Label: "Менеджеры", Href: "/panel/managers", Active: strings.HasPrefix(path, "/managers")})
	}

	if (permissions.CanViewAuditLog()) {
		items = append(items, templates.NavItem{Label: "Логи", Href: "/panel/audit", Active: path == "/audit"})
	}

	if (permissions.CanViewOwnProfile()) {
		items = append(items, templates.NavItem{Label: "Мой профиль", Href: "/panel/profile", Active: path == "/profile"})
	}

	items = append(items, templates.NavItem{Label: "Настройки", Href: "/panel/settings", Active: strings.HasPrefix(path, "/settings")})

	return items
}
