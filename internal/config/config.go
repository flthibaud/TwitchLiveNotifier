package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds configuration values for the bot
type Config struct {
	Port                 string // Port for the HTTP server
	BotToken             string // Discord bot token
	TwitchClientID       string // Twitch application client ID
	TwitchClientSecret   string // Twitch application client secret
	TwitchWebhookSecret  string // Twitch webhook secret
	TwitchBroadcasterIDs []string
	CallbackURL          string // URL for Twitch webhook callback
	NotifyChannelID      string // Default Discord channel ID for notifications
}

// Load reads configuration from environment variables (and .env file) and returns a Config
func Load() (*Config, error) {
	// Load .env in development if present
	_ = godotenv.Load()

	cfg := &Config{
		Port:                os.Getenv("PORT"),
		BotToken:            os.Getenv("BOT_TOKEN"),
		TwitchClientID:      os.Getenv("TWITCH_CLIENT_ID"),
		TwitchClientSecret:  os.Getenv("TWITCH_CLIENT_SECRET"),
		TwitchWebhookSecret: os.Getenv("TWITCH_WEBHOOK_SECRET"),
		CallbackURL:         os.Getenv("CALLBACK_URL"),
		NotifyChannelID:     os.Getenv("NOTIFY_CHANNEL_ID"),
	}

	// Validate required fields
	missing := []string{}
	if cfg.Port == "" {
		missing = append(missing, "PORT")
	}
	if cfg.BotToken == "" {
		missing = append(missing, "BOT_TOKEN")
	}
	if cfg.TwitchClientID == "" {
		missing = append(missing, "TWITCH_CLIENT_ID")
	}
	if cfg.TwitchClientSecret == "" {
		missing = append(missing, "TWITCH_CLIENT_SECRET")
	}
	if cfg.TwitchWebhookSecret == "" {
		missing = append(missing, "TWITCH_WEBHOOK_SECRET")
	}
	if cfg.CallbackURL == "" {
		missing = append(missing, "CALLBACK_URL")
	}
	idsEnv := os.Getenv("TWITCH_BROADCASTER_IDS")
	if idsEnv != "" {
		cfg.TwitchBroadcasterIDs = strings.Split(idsEnv, ",")
	}
	if cfg.NotifyChannelID == "" {
		missing = append(missing, "NOTIFY_CHANNEL_ID")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}
