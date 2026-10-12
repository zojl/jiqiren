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

		r.Post("/target", handlers.SetTarget)

		r.Get("/squads/new", handlers.NewSquadForm)
		r.Post("/squads", handlers.CreateSquad)
		r.Get("/squads/{id}/rename", handlers.RenameSquadForm)
		r.Post("/squads/{id}/rename", handlers.RenameSquad)
		r.Post("/squads/{id}/delete", handlers.DeleteSquad)

		r.Get("/squads/{id}/token", handlers.SquadTokenForm)
		r.Post("/squads/{id}/token", handlers.SetSquadToken)

		r.Get("/fraction/token", handlers.FractionTokenForm)
		r.Post("/fraction/token", handlers.SetFractionToken)
		r.Get("/fraction/budget", handlers.FractionBudget)

		r.Get("/managers", handlers.ManagerList)
		r.Post("/managers", handlers.CreateManager)
		r.Get("/managers/{id}/edit", handlers.EditManagerForm)
		r.Post("/managers/{id}/edit", handlers.EditManager)
		r.Post("/managers/{id}/delete", handlers.DeleteManager)
	})

	return r
}
