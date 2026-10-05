package config

import "testing"

func TestLoadDefaultHTTPAddr(t *testing.T) {
	t.Setenv("CUVOTE_HTTP_ADDR", "")
	t.Setenv("PORT", "")

	cfg := Load()

	if cfg.HTTPAddr != ":8080" {
		t.Fatalf(
			"expected HTTP address %q, got %q",
			":8080",
			cfg.HTTPAddr,
		)
	}
}

func TestLoadCustomHTTPAddr(t *testing.T) {
	t.Setenv("CUVOTE_HTTP_ADDR", ":3000")
	t.Setenv("PORT", "")

	cfg := Load()

	if cfg.HTTPAddr != ":3000" {
		t.Fatalf(
			"expected HTTP address %q, got %q",
			":3000",
			cfg.HTTPAddr,
		)
	}
}

func TestLoadPortOverridesHTTPAddr(t *testing.T) {
	t.Setenv("CUVOTE_HTTP_ADDR", ":3000")
	t.Setenv("PORT", "9000")

	cfg := Load()

	if cfg.HTTPAddr != ":9000" {
		t.Fatalf(
			"expected HTTP address %q, got %q",
			":9000",
			cfg.HTTPAddr,
		)
	}
}
