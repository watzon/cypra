package hostedlogin

import "testing"

func TestValidateAccentRejectsFailingContrast(t *testing.T) {
	err := ValidateAccent("#FFFF00")
	if err == nil {
		t.Fatal("expected failing accent")
	}
	validation, ok := err.(AccentValidationError)
	if !ok {
		t.Fatalf("expected AccentValidationError, got %T", err)
	}
	if validation.Pair == "" || validation.Ratio <= 0 {
		t.Fatalf("missing validation details: %#v", validation)
	}
}

func TestThemeKeepsSystemFocusRingSeparate(t *testing.T) {
	theme, err := ThemeFromJSON("acme", "Acme", []byte(`{"accent":"#767676","display_name":"Acme Login"}`), "")
	if err != nil {
		t.Fatal(err)
	}
	if theme.Accent == LightFocus || theme.Accent == DarkFocus {
		t.Fatalf("test setup must use tenant accent distinct from focus tokens")
	}
	if LightFocus != "#0D9488" || DarkFocus != "#2DD4BF" {
		t.Fatalf("system focus tokens changed")
	}
}
