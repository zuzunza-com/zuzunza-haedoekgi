package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const EnvServgateURL = "ZUZUNZA_SERVGATE_URL"

type Settings struct {
	ServgateURL string
	OutputDir   string
}

func Load(outputFlag string) (Settings, error) {
	base := strings.TrimSpace(os.Getenv(EnvServgateURL))
	if base == "" {
		return Settings{}, fmt.Errorf("%s 환경 변수가 필요합니다 (예: https://www.zuzunza.com/xpi)", EnvServgateURL)
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
