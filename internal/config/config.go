package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	EnvServgateURL     = "ZUZUNZA_SERVGATE_URL"
	DefaultServgateURL = "https://www.zuzunza.com/xpi"
)

type Settings struct {
	ServgateURL string
	OutputDir   string
}

func Load(outputFlag string) (Settings, error) {
	base := strings.TrimSpace(os.Getenv(EnvServgateURL))
	if base == "" {
		base = DefaultServgateURL
	}
	base = strings.TrimRight(base, "/")

	out := strings.TrimSpace(outputFlag)
	if out == "" {
		out = "."
	}
	abs, err := filepath.Abs(out)
	if err != nil {
		return Settings{}, err
	}
	return Settings{ServgateURL: base, OutputDir: abs}, nil
}
