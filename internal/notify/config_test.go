package notify

import "testing"

func TestConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("SMTP_USER", "sender@gmail.com")
	t.Setenv("SMTP_PASS", "apppass")
	t.Setenv("NOTIFY_TO", "")
	t.Setenv("NOTIFY_THRESHOLD", "")

	cfg := ConfigFromEnv()
	if cfg.User != "sender@gmail.com" || cfg.Password != "apppass" {
		t.Fatalf("user/password not read: %+v", cfg)
	}
	if cfg.Recipient != "sender@gmail.com" {
		t.Fatalf("recipient should default to user, got %q", cfg.Recipient)
	}
	if cfg.Threshold != 20 {
		t.Fatalf("threshold should default to 20, got %v", cfg.Threshold)
	}
	if !cfg.Configured() {
		t.Fatal("expected configured")
	}
}

func TestConfigFromEnvInvalidThreshold(t *testing.T) {
	t.Setenv("SMTP_USER", "sender@gmail.com")
	t.Setenv("SMTP_PASS", "apppass")
	for _, v := range []string{"abc", "0", "-5"} {
		t.Setenv("NOTIFY_THRESHOLD", v)
		if got := ConfigFromEnv().Threshold; got != 20 {
			t.Fatalf("threshold %q should fallback to 20, got %v", v, got)
		}
	}
}

func TestConfigNotConfiguredWithoutCredentials(t *testing.T) {
	t.Setenv("SMTP_USER", "")
	t.Setenv("SMTP_PASS", "")
	if ConfigFromEnv().Configured() {
		t.Fatal("expected not configured")
	}
}
