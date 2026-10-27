package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ListenAddr    string
	DatabaseURL   string
	StorageDir    string
	BotAPIURL     string
	BotAPIToken   string
	WAHashKey     string
	AdminUsername string
	AdminPassword string
	CookieSecure  bool
	RetentionDays int
	MaxAttachment int64
}

func Load() (Config, error) {
	cfg := Config{
		ListenAddr:    value("HTTP_ADDR", ":8080"),
		DatabaseURL:   strings.TrimSpace(os.Getenv("DATABASE_URL")),
		StorageDir:    strings.TrimSpace(os.Getenv("STORAGE_DIR")),
		BotAPIURL:     value("BOT_API_URL", "http://app:8080"),
		BotAPIToken:   os.Getenv("BOT_API_TOKEN"),
		WAHashKey:     os.Getenv("WA_HASH_KEY"),
		AdminUsername: strings.TrimSpace(os.Getenv("ADMIN_USERNAME")),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		MaxAttachment: 10 << 20,
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

	var err error
	cfg.CookieSecure, err = strconv.ParseBool(value("SESSION_COOKIE_SECURE", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("SESSION_COOKIE_SECURE: %w", err)
	}
	cfg.RetentionDays, err = strconv.Atoi(strings.TrimSpace(os.Getenv("ATTACHMENT_RETENTION_DAYS")))
	if err != nil || cfg.RetentionDays < 1 || cfg.RetentionDays > 3650 {
		return Config{}, errors.New("ATTACHMENT_RETENTION_DAYS must be between 1 and 3650 and approved by the village")
	}
	return cfg, nil
}

func value(name, fallback string) string {
	if got := strings.TrimSpace(os.Getenv(name)); got != "" {
		return got
	}
	return fallback
}
