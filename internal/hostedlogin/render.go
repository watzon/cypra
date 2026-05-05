// Package hostedlogin renders server-owned authentication pages.
package hostedlogin

import (
	"embed"
	"html/template"
	"io"
	"path/filepath"
	"strings"
)

//go:embed templates/*.html static/*.js
var files embed.FS

// PageData is the template model shared by hosted-login pages.
type PageData struct {
	Title       string
	Route       string
	Theme       Theme
	Error       string
	Message     string
	Email       string
	ContinueURL string
	Scopes      []Scope
	RPID        string
}

// Scope describes one OIDC consent permission.
type Scope struct {
	Name        string
	Description string
	New         bool
}

// Renderer executes the embedded hosted-login templates.
type Renderer struct {
	templates *template.Template
}

// NewRenderer parses the embedded template set.
func NewRenderer() (Renderer, error) {
	tpl, err := template.ParseFS(files, "templates/base.html")
	if err != nil {
		return Renderer{}, err
	}
	return Renderer{templates: tpl}, nil
}

// Render writes a named hosted-login page.
func (r Renderer) Render(w io.Writer, name string, data PageData) error {
	if strings.TrimSpace(data.Theme.Accent) == "" {
		data.Theme.Accent = "#0D9488"
	}
	tpl, err := r.templates.Clone()
	if err != nil {
		return err
	}
	if _, err := tpl.ParseFS(files, filepath.Join("templates", name+".html")); err != nil {
		return err
	}
	return tpl.ExecuteTemplate(w, name+".html", data)
}

// PasskeyJS returns the embedded vanilla passkey helper.
func PasskeyJS() ([]byte, error) {
	return files.ReadFile("static/passkey.js")
}
