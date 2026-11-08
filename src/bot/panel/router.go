package panel

import (
	"net/http"

	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/handlers"
	"jiqiren/bot/panel/static"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/go-chi/chi/v5"
)

func SetVK(client *api.VK) {
	handlers.SetVK(client)
}

func NewRouter() chi.Router {
	r := chi.NewRouter()

	r.Handle("/static/*", http.StripPrefix("/static", http.FileServerFS(static.Files)))

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth)

		r.Get("/", handlers.Dashboard)

		r.Post("/logout", handlers.Logout)

		r.Post("/target", handlers.SetTarget)

		r.Get("/squads/new", handlers.NewSquadForm)
		r.Post("/squads", handlers.CreateSquad)
		r.Get("/squads/{code}", handlers.SquadPage)
		r.Get("/squads/{code}/rename", handlers.RenameSquadForm)
		r.Post("/squads/{code}/rename", handlers.RenameSquad)
		r.Post("/squads/{code}/delete", handlers.DeleteSquad)

		r.Get("/squads/{code}/token", handlers.SquadTokenForm)
		r.Post("/squads/{code}/token", handlers.SetSquadToken)
		r.Post("/squads/{code}/unbind-chat", handlers.UnbindSquadChat)

		r.Post("/squads/{code}/members", handlers.AddSquadMemberOverride)
		r.Post("/squads/{code}/members/{vk_id}/remove", handlers.RemoveSquadMemberOverride)
		r.Post("/squads/{code}/members/{vk_id}/freeze", handlers.FreezeSquadMember)

		r.Get("/squads/{code}/schedule", handlers.SquadSchedule)
		r.Get("/squads/{code}/schedule/new", handlers.NewScheduledMessageForm)
		r.Post("/squads/{code}/schedule", handlers.CreateScheduledMessage)
		r.Get("/squads/{code}/schedule/{id}/edit", handlers.EditScheduledMessageForm)
		r.Post("/squads/{code}/schedule/{id}/edit", handlers.EditScheduledMessage)
		r.Post("/squads/{code}/schedule/{id}/delete", handlers.DeleteScheduledMessage)

		r.Get("/squads/{code}/targets", handlers.SquadPlannedTargetList)
		r.Get("/squads/{code}/targets/new", handlers.NewSquadPlannedTargetForm)
		r.Post("/squads/{code}/targets", handlers.CreateSquadPlannedTarget)
		r.Get("/squads/{code}/targets/{id}/edit", handlers.EditSquadPlannedTargetForm)
		r.Post("/squads/{code}/targets/{id}/edit", handlers.EditSquadPlannedTarget)
		r.Post("/squads/{code}/targets/{id}/delete", handlers.DeleteSquadPlannedTarget)

		r.Get("/players/{vk_id}", handlers.PlayerPage)

		r.Get("/fraction", handlers.FractionPage)
		r.Get("/fraction/token", handlers.FractionTokenForm)
		r.Post("/fraction/token", handlers.SetFractionToken)
		r.Get("/fraction/message", handlers.FractionMessageForm)
		r.Post("/fraction/message", handlers.SetFractionMessage)
		r.Get("/fraction/settings", handlers.FractionSettingsForm)
		r.Post("/fraction/settings", handlers.SetFractionReadinessSettings)
		r.Get("/fraction/budget", handlers.BudgetPage)
		r.Get("/fraction/budget/actual", handlers.BudgetActual)
		r.Get("/fraction/budget/chart", handlers.BudgetChart)

		r.Get("/fraction/targets", handlers.FractionPlannedTargetList)
		r.Get("/fraction/targets/new", handlers.NewFractionPlannedTargetForm)
		r.Post("/fraction/targets", handlers.CreateFractionPlannedTarget)
		r.Get("/fraction/targets/{id}/edit", handlers.EditFractionPlannedTargetForm)
		r.Post("/fraction/targets/{id}/edit", handlers.EditFractionPlannedTarget)
		r.Post("/fraction/targets/{id}/delete", handlers.DeleteFractionPlannedTarget)

		r.Get("/managers", handlers.ManagerList)
		r.Post("/managers", handlers.CreateManager)
		r.Get("/managers/{vk_id}/edit", handlers.EditManagerForm)
		r.Post("/managers/{vk_id}/edit", handlers.EditManager)
		r.Post("/managers/{vk_id}/delete", handlers.DeleteManager)

		r.Get("/audit", handlers.AuditLog)

		r.Get("/settings/theme", handlers.ThemeSettingsForm)
		r.Post("/settings/theme", handlers.SetManagerTheme)
	})

	return r
}
