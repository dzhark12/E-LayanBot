package config

import (
	"strings"
	"testing"
)

func setValidEnvironment(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{
		"DATABASE_URL": "postgres://test", "STORAGE_DIR": t.TempDir(),
		"BOT_API_TOKEN": strings.Repeat("b", 32), "WA_HASH_KEY": strings.Repeat("h", 32),
		"ADMIN_USERNAME": "admin", "ADMIN_PASSWORD": strings.Repeat("p", 12),
		"ATTACHMENT_RETENTION_DAYS": "30",
	} {
		t.Setenv(key, value)
	}
	for _, key := range []string{"HTTP_ADDR", "BOT_API_URL", "SESSION_COOKIE_SECURE", "GEMINI_API_KEY", "GEMINI_MODEL", "VILLAGE_NAME", "GEMINI_MAX_PER_HOUR"} {
		t.Setenv(key, "")
	}
}

func TestLoadDefaults(t *testing.T) {
	setValidEnvironment(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr != ":8080" || cfg.BotAPIURL != "http://app:8080" || cfg.GeminiModel != "gemini-3.5-flash-lite" || cfg.GeminiPerHour != 10 || cfg.RetentionDays != 30 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadRejectsInvalidRequiredSettings(t *testing.T) {
	for _, tc := range []struct{ key, value, want string }{
		{"DATABASE_URL", "", "DATABASE_URL is required"},
		{"BOT_API_TOKEN", "short", "BOT_API_TOKEN must contain"},
		{"WA_HASH_KEY", "short", "WA_HASH_KEY must contain"},
		{"ADMIN_PASSWORD", "short", "ADMIN_PASSWORD must contain"},
		{"ATTACHMENT_RETENTION_DAYS", "0", "ATTACHMENT_RETENTION_DAYS"},
		{"GEMINI_MAX_PER_HOUR", "201", "GEMINI_MAX_PER_HOUR"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			setValidEnvironment(t)
			t.Setenv(tc.key, tc.value)
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Load() error = %v, want containing %q", err, tc.want)
			}
		})
	}
}
