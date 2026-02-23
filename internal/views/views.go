// Package views renders html templates to create an HTML ui
package views

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
)

//go:embed *.gohtml
var files embed.FS

var tmpl = map[string]*template.Template{}

func init() {
	for _, page := range []string{
		"login.gohtml",
		"user_list.gohtml",
		"users_form.gohtml",
	} {
		tmpl[page] = template.Must(template.ParseFS(files, "base.gohtml", page))
	}
}

func Render(w http.ResponseWriter, page string, data any) error {
	t, ok := tmpl[page]
	if !ok {
		return fmt.Errorf("template %q not found", page)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return t.ExecuteTemplate(w, "base.gohtml", data)
}
