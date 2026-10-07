package config

import (
	"os"
)

const defaultHTTPAddr = ":8080"

type Config struct {
	HTTPAddr    string
	DatabaseURL string
}

func Load() Config {
	addr := os.Getenv("CUVOTE_HTTP_ADDR")
	if addr == "" {
		addr = defaultHTTPAddr
	}

	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	return Config{
		HTTPAddr:    addr,
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
}
