package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ListenAddr        string
	DatabaseURL       string
	StorageDir        string
	BotAPIURL         string
	BotAPIToken       string
	WAHashKey         string
	AdminUsername     string
	AdminPassword     string
	AdminRecoveryCode string
	CookieSecure      bool
	RetentionDays     int
	MaxAttachment     int64

	GeminiAPIKey  string
	GeminiModel   string
	GeminiPerHour int
	VillageName   string

	AdminEmail   string
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	AppBaseURL   string
}

func Load() (Config, error) {
	cfg := Config{
<<<<<<< HEAD
		ListenAddr:    value("HTTP_ADDR", ":8080"),
		DatabaseURL:   strings.TrimSpace(os.Getenv("DATABASE_URL")),
		StorageDir:    strings.TrimSpace(os.Getenv("STORAGE_DIR")),
		BotAPIURL:     value("BOT_API_URL", "http://app:8080"),
		BotAPIToken:   os.Getenv("BOT_API_TOKEN"),
		WAHashKey:     os.Getenv("WA_HASH_KEY"),
		AdminUsername: strings.TrimSpace(os.Getenv("ADMIN_USERNAME")),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		MaxAttachment: 10 << 20,
		GeminiAPIKey:  strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
		GeminiModel:   value("GEMINI_MODEL", "gemini-3.5-flash-lite"),
		VillageName:   value("VILLAGE_NAME", "Sabah Balau"),
		AdminEmail:   strings.TrimSpace(os.Getenv("ADMIN_EMAIL")),
		SMTPHost:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPUsername: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:     value("SMTP_FROM", "E-LAYAN Desa Sabah Balau <no-reply@sabahbalau.desa.id>"),
		AppBaseURL:   strings.TrimRight(value("APP_BASE_URL", "http://localhost:8080"), "/"),
=======
		ListenAddr:        value("HTTP_ADDR", ":8080"),
		DatabaseURL:       strings.TrimSpace(os.Getenv("DATABASE_URL")),
		StorageDir:        strings.TrimSpace(os.Getenv("STORAGE_DIR")),
		BotAPIURL:         value("BOT_API_URL", "http://app:8080"),
		BotAPIToken:       os.Getenv("BOT_API_TOKEN"),
		WAHashKey:         os.Getenv("WA_HASH_KEY"),
		AdminUsername:     strings.TrimSpace(os.Getenv("ADMIN_USERNAME")),
		AdminPassword:     os.Getenv("ADMIN_PASSWORD"),
		AdminRecoveryCode: strings.TrimSpace(os.Getenv("ADMIN_RECOVERY_CODE")),
		MaxAttachment:     10 << 20,
		GeminiAPIKey:      strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
		GeminiModel:       value("GEMINI_MODEL", "gemini-3.5-flash-lite"),
		VillageName:       value("VILLAGE_NAME", "Sabah Balau"),
>>>>>>> 104a744aa7baf8b17dc213d3aa67c3623eff856b
	}

	for name, got := range map[string]string{
		"DATABASE_URL":   cfg.DatabaseURL,
		"STORAGE_DIR":    cfg.StorageDir,
		"BOT_API_TOKEN":  cfg.BotAPIToken,
		"WA_HASH_KEY":    cfg.WAHashKey,
		"ADMIN_USERNAME": cfg.AdminUsername,
		"ADMIN_PASSWORD": cfg.AdminPassword,
	} {
		if got == "" {
			return Config{}, fmt.Errorf("%s is required", name)
		}
	}
	if len(cfg.BotAPIToken) < 32 {
		return Config{}, errors.New("BOT_API_TOKEN must contain at least 32 characters")
	}
	if len(cfg.WAHashKey) < 32 {
		return Config{}, errors.New("WA_HASH_KEY must contain at least 32 characters")
	}
	if len(cfg.AdminPassword) < 12 {
		return Config{}, errors.New("ADMIN_PASSWORD must contain at least 12 characters")
	}
	if cfg.AdminRecoveryCode != "" && len(cfg.AdminRecoveryCode) < 32 {
		return Config{}, errors.New("ADMIN_RECOVERY_CODE must contain at least 32 characters")
	}

	var err error
	cfg.CookieSecure, err = strconv.ParseBool(value("SESSION_COOKIE_SECURE", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("SESSION_COOKIE_SECURE: %w", err)
	}
	cfg.RetentionDays, err = strconv.Atoi(strings.TrimSpace(os.Getenv("ATTACHMENT_RETENTION_DAYS")))
	if err != nil || cfg.RetentionDays < 1 || cfg.RetentionDays > 3650 {
		return Config{}, errors.New("ATTACHMENT_RETENTION_DAYS must be between 1 and 3650 and approved by the village")
	}
	cfg.SMTPPort = 587
	if raw := strings.TrimSpace(os.Getenv("SMTP_PORT")); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, errors.New("SMTP_PORT must be a valid port number between 1 and 65535")
		}
		cfg.SMTPPort = port
	}
	cfg.GeminiPerHour = 10
	if raw := strings.TrimSpace(os.Getenv("GEMINI_MAX_PER_HOUR")); raw != "" {
		cfg.GeminiPerHour, err = strconv.Atoi(raw)
		if err != nil || cfg.GeminiPerHour < 1 || cfg.GeminiPerHour > 200 {
			return Config{}, errors.New("GEMINI_MAX_PER_HOUR must be between 1 and 200")
		}
	}
	return cfg, nil
}

func value(name, fallback string) string {
	if got := strings.TrimSpace(os.Getenv(name)); got != "" {
		return got
	}
	return fallback
}
