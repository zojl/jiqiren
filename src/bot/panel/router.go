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

		r.Get("/squads/{code}/schedule", handlers.SquadSchedule)
		r.Get("/squads/{code}/schedule/new", handlers.NewScheduledMessageForm)
		r.Post("/squads/{code}/schedule", handlers.CreateScheduledMessage)
		r.Get("/squads/{code}/schedule/{id}/edit", handlers.EditScheduledMessageForm)
		r.Post("/squads/{code}/schedule/{id}/edit", handlers.EditScheduledMessage)
		r.Post("/squads/{code}/schedule/{id}/delete", handlers.DeleteScheduledMessage)

		r.Get("/fraction/token", handlers.FractionTokenForm)
		r.Post("/fraction/token", handlers.SetFractionToken)
		r.Get("/fraction/budget", handlers.FractionBudget)

		r.Get("/managers", handlers.ManagerList)
		r.Post("/managers", handlers.CreateManager)
		r.Get("/managers/{vk_id}/edit", handlers.EditManagerForm)
		r.Post("/managers/{vk_id}/edit", handlers.EditManager)
		r.Post("/managers/{vk_id}/delete", handlers.DeleteManager)

		r.Get("/audit", handlers.AuditLog)
	})

	return r
}
