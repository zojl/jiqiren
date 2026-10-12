package handlers

import (
	"net/http"

	"jiqiren/bot/panel/auth"
)

func Logout(w http.ResponseWriter, r *http.Request) {
	auth.ClearSession(w)
	http.Redirect(w, r, "/", http.StatusFound)
}
