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
	Title            string
	Route            string
	Theme            Theme
	Error            string
	Message          string
	Email            string
	ContinueURL      string
	Scopes           []Scope
	RPID             string
	InviteID         string
	ResetToken       string
	GoogleURL        string
	PasskeyEnabled   bool
	PasswordEnabled  bool
	MagicLinkEnabled bool
	GoogleEnabled    bool
	SignupEnabled    bool
	AnyMethodEnabled bool
	// PrimaryMethod is the tenant-chosen default sign-in method (or, when no
	// default is set, the first enabled method by static priority). The login
	// template uses it to decide which form to lead with.
	//
	// Built-in identifiers use their short name (password / magic_link /
	// passkey). Social SSO connections use "social:<kind>" (e.g. "social:google",
	// "social:microsoft"); enterprise OIDC connections use "oidc:<slug>".
	PrimaryMethod string
	// SocialButtons is one entry per enabled social_connections row.
	SocialButtons []SocialButton
	// OIDCButtons is one entry per enabled oidc_connections row.
	OIDCButtons []OIDCButton
}

// SocialButton describes one enabled social SSO connection rendered by the
// hosted-login template.
type SocialButton struct {
	Kind    string
	Label   string
	IconURL string
	URL     string
}

// OIDCButton describes one enabled enterprise OIDC connection rendered by the
// hosted-login template.
type OIDCButton struct {
	Slug        string
	DisplayName string
	URL         string
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
	tpl, err := template.New("hostedlogin").Funcs(templateFuncs()).ParseFS(files, "templates/base.html")
	if err != nil {
		return Renderer{}, err
	}
	return Renderer{templates: tpl}, nil
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"hasPrefix":  strings.HasPrefix,
		"trimPrefix": strings.TrimPrefix,
	}
}

// Render writes a named hosted-login page.
func (r Renderer) Render(w io.Writer, name string, data PageData) error {
	if strings.TrimSpace(data.Theme.Accent) == "" {
		data.Theme.Accent = "#0F766E"
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

// InstanceAdminLoginJS returns the embedded instance-admin login helper.
func InstanceAdminLoginJS() ([]byte, error) {
	return files.ReadFile("static/instance-admin-login.js")
}
