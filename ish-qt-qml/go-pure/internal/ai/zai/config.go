package zai

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	APIKey  string
	BaseURL string
	Model   string
}

func DefaultConfig() Config {
	return Config{BaseURL: "https://api.z.ai/api/paas/v4/", Model: "glm-5.3"}
}

func ConfigFromEnv() Config {
	cfg := DefaultConfig()
	cfg.APIKey = os.Getenv("ZAI_API_KEY")
	if value := os.Getenv("ZAI_BASE_URL"); value != "" {
		cfg.BaseURL = value
	}
	if value := os.Getenv("ZAI_MODEL"); value != "" {
		cfg.Model = value
	}
	return cfg
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.APIKey) == "" {
		return errors.New("zai: ZAI_API_KEY is required")
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return errors.New("zai: base URL is required")
	}
	if strings.TrimSpace(c.Model) == "" {
		return errors.New("zai: model is required")
	}
	return nil
}
