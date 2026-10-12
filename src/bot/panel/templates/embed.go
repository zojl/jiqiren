package templates

import (
	"bytes"
	"embed"
	"html/template"
	"net/http"
)

//go:embed *.html.tmpl
var files embed.FS

var parsed = template.Must(template.ParseFS(files, "*.html.tmpl"))

// Render executes the named page template, then wraps its output in the
// shared layout. Page templates must each use a unique {{define}} name
// (layout.html.tmpl is the only template named "layout").
func Render(w http.ResponseWriter, name string, data interface{}) {
	var body bytes.Buffer
	if err := parsed.ExecuteTemplate(&body, name, data); err != nil {
		http.Error(w, "Ошибка шаблона: "+err.Error(), http.StatusInternalServerError)
		return
	}

	layoutData := struct{ Body template.HTML }{Body: template.HTML(body.String())}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := parsed.ExecuteTemplate(w, "layout", layoutData); err != nil {
		http.Error(w, "Ошибка шаблона: "+err.Error(), http.StatusInternalServerError)
	}
}
