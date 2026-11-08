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

// NavItem is one entry of the permission-filtered top navigation, built
// fresh for every request by BuildNav.
type NavItem struct {
	Label  string
	Href   string
	Active bool
}

// BuildNav is set by the handlers package at init time — templates stays
// free of auth/database imports, while Render can still inject the nav on
// every page without every call site building it by hand.
var BuildNav func(r *http.Request) []NavItem

// Theme is the resolved data-mode/data-scheme pair for the <html> tag.
type Theme struct {
	Mode   string
	Scheme string
}

// BuildTheme is set by the handlers package at init time, mirroring BuildNav
// — it resolves the current manager's theme preference (default on a nil/
// pre-auth manager) without templates importing auth/database itself.
var BuildTheme func(r *http.Request) Theme

// Render executes the named page template, then wraps its output in the
// shared layout together with the request's nav. Page templates must each
// use a unique {{define}} name (layout.html.tmpl is the only template
// named "layout").
func Render(w http.ResponseWriter, r *http.Request, name string, data interface{}) {
	var body bytes.Buffer
	if err := parsed.ExecuteTemplate(&body, name, data); err != nil {
		http.Error(w, "Ошибка шаблона: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var nav []NavItem
	if (BuildNav != nil) {
		nav = BuildNav(r)
	}

	theme := Theme{Mode: "night", Scheme: "hacker"}
	if (BuildTheme != nil) {
		theme = BuildTheme(r)
	}

	layoutData := struct {
		Body  template.HTML
		Nav   []NavItem
		Theme Theme
	}{Body: template.HTML(body.String()), Nav: nav, Theme: theme}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := parsed.ExecuteTemplate(w, "layout", layoutData); err != nil {
		http.Error(w, "Ошибка шаблона: "+err.Error(), http.StatusInternalServerError)
	}
}

// RenderFragment executes the named template on its own, with no layout
// wrapper — for small HTML snippets fetched by JS (e.g. a lazily loaded
// spoiler) rather than full pages.
func RenderFragment(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := parsed.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Ошибка шаблона: "+err.Error(), http.StatusInternalServerError)
	}
}
