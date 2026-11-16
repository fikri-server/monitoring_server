package notify

import (
	"math"
	"os"
	"strconv"
)

const defaultThreshold = 20

type Config struct {
	User      string
	Password  string
	Recipient string
	Threshold float64
}

func ConfigFromEnv() Config {
	cfg := Config{
		User:     os.Getenv("SMTP_USER"),
		Password: os.Getenv("SMTP_PASS"),
	}
	cfg.Recipient = os.Getenv("NOTIFY_TO")
	if cfg.Recipient == "" {
		cfg.Recipient = cfg.User
	}

	cfg.Threshold = defaultThreshold
	if v, err := strconv.ParseFloat(os.Getenv("NOTIFY_THRESHOLD"), 64); err == nil && v > 0 && !math.IsNaN(v) {
		cfg.Threshold = v
	}
	return cfg
}

func (c Config) Configured() bool {
	return c.User != "" && c.Password != ""
}
