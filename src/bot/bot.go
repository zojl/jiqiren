package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "strings"

    "jiqiren/bot/messageHandler"
    "jiqiren/bot/database"
    "jiqiren/bot/panel"
    "jiqiren/bot/panel/auth"

    "github.com/SevereCloud/vksdk/v2/api"
    "github.com/SevereCloud/vksdk/v2/events"
    "github.com/SevereCloud/vksdk/v2/callback"
)

func main() {
    token := os.Getenv("TOKEN")
    prefix := os.Getenv("PREFIX")
    vk := api.NewVK(token)

    database.Init()

    // get information about the group
    _, err := vk.GroupsGetByID(nil)
    if err != nil {
        log.Fatal(err)
    }

    cb := callback.NewCallback()
    cb.ConfirmationKey = os.Getenv("CALLBACK_RESPONSE")
    cb.SecretKey = os.Getenv("CALLBACK_SECRET")

    cb.MessageNew(func(_ context.Context, obj events.MessageNewObject) {
        if (os.Getenv("ENV") == "dev") {
            log.Printf("%+v\n", obj)
        }

        if (strings.HasPrefix(obj.Message.Text, prefix)) {
            messageHandler.HandleCommand(&obj, vk)
        } else {
            messageHandler.HandlePlain(obj, vk)
        }
    })

    mux := http.NewServeMux()
    mux.Handle("/panel/", http.StripPrefix("/panel", panel.NewRouter()))

    callbackPath := os.Getenv("CALLBACK_URL")
    mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
        if (r.Method == http.MethodGet) {
            if code := r.URL.Query().Get("login"); code != "" {
                auth.HandleLoginRedirect(w, r, code)
                return
            }

            if (auth.HasValidSession(r)) {
                http.Redirect(w, r, "/panel", http.StatusFound)
            } else {
                w.Header().Set("Content-Type", "text/plain; charset=utf-8")
                w.Write([]byte("Войдите через бота: команда /login в личных сообщениях."))
            }
            return
        }

        cb.HandleFunc(w, r)
    })

    log.Println("Starting Web Server")
    if err := http.ListenAndServe(":" + os.Getenv("CALLBACK_PORT"), mux); err != nil {
        log.Fatal(err)
    }
}
