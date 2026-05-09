package hostedlogin

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// System token values that tenant branding cannot override.
const (
	DarkCanvas        = "#0A0A0B"
	LightCanvas       = "#FAFAFA"
	DarkTextOnAccent  = "#FFFFFF"
	LightTextOnAccent = "#FFFFFF"
	DarkFocus         = "#2DD4BF"
	LightFocus        = "#0D9488"
)

// Branding is the tenant-controlled hosted-login branding payload.
type Branding struct {
	LogoURL     string `json:"logo_url"`
	Accent      string `json:"accent"`
	DisplayName string `json:"display_name"`
	PoweredBy   *bool  `json:"powered_by"`
}

// Theme is the resolved hosted-login theme emitted into templates.
type Theme struct {
	TenantSlug  string
	LogoURL     string
	Accent      string
	DisplayName string
	PoweredBy   bool
}

// AccentValidationError reports the failed contrast pair and measured ratio.
type AccentValidationError struct {
	Pair  string
	Ratio float64
}

func (e AccentValidationError) Error() string {
	return fmt.Sprintf("accent contrast failed for %s: %.2f", e.Pair, e.Ratio)
}

// ThemeFromJSON resolves tenant branding JSON into a safe template theme.
// When uploadedLogoURL is non-empty it overrides any logo_url stored in the
// branding JSONB — this is how an uploaded tenant logo wins over a manual
// CDN URL set via the branding PATCH.
func ThemeFromJSON(slug, tenantName string, raw []byte, uploadedLogoURL string) (Theme, error) {
	branding := Branding{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &branding); err != nil {
			return Theme{}, err
		}
	}
	poweredBy := true
	if branding.PoweredBy != nil {
		poweredBy = *branding.PoweredBy
	}
	displayName := strings.TrimSpace(branding.DisplayName)
	if displayName == "" {
		displayName = tenantName
	}
	accent := strings.TrimSpace(branding.Accent)
	customAccent := accent != ""
	if accent == "" {
		accent = "#0F766E"
	}
	if customAccent {
		if err := ValidateAccent(accent); err != nil {
			return Theme{}, err
		}
	}
	logoURL := branding.LogoURL
	if uploadedLogoURL != "" {
		logoURL = uploadedLogoURL
	}
	return Theme{TenantSlug: slug, LogoURL: logoURL, Accent: accent, DisplayName: displayName, PoweredBy: poweredBy}, nil
}

// ValidateAccent enforces the hosted-login tenant accent contrast gate.
func ValidateAccent(accent string) error {
	if _, _, _, err := parseHexColor(accent); err != nil {
		return err
	}
	pairs := []struct {
		name string
		fg   string
		bg   string
		min  float64
	}{
		{name: "text-on-accent-light", fg: LightTextOnAccent, bg: accent, min: 4.5},
		{name: "accent-on-dark-canvas", fg: accent, bg: DarkCanvas, min: 3},
		{name: "accent-on-light-canvas", fg: accent, bg: LightCanvas, min: 3},
	}
	for _, pair := range pairs {
		ratio, err := ContrastRatio(pair.fg, pair.bg)
		if err != nil {
			return err
		}
		if ratio < pair.min {
			return AccentValidationError{Pair: pair.name, Ratio: ratio}
		}
	}
	return nil
}

// ContrastRatio computes the WCAG contrast ratio between two hex colors.
func ContrastRatio(fg, bg string) (float64, error) {
	fr, fgG, fb, err := parseHexColor(fg)
	if err != nil {
		return 0, err
	}
	br, bgG, bb, err := parseHexColor(bg)
	if err != nil {
		return 0, err
	}
	l1 := relativeLuminance(fr, fgG, fb)
	l2 := relativeLuminance(br, bgG, bb)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05), nil
}

func parseHexColor(value string) (float64, float64, float64, error) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(value) != 6 {
		return 0, 0, 0, fmt.Errorf("accent must be #RRGGBB")
	}
	parsed, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("accent must be #RRGGBB")
	}
	return float64((parsed>>16)&0xff) / 255, float64((parsed>>8)&0xff) / 255, float64(parsed&0xff) / 255, nil
}

func relativeLuminance(r, g, b float64) float64 {
	convert := func(v float64) float64 {
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*convert(r) + 0.7152*convert(g) + 0.0722*convert(b)
}
