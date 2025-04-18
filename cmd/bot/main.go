package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flthibaud/TwitchLiveNotifier/internal/config"
	"github.com/flthibaud/TwitchLiveNotifier/internal/discord"
	"github.com/flthibaud/TwitchLiveNotifier/internal/discord/twitch"
	"github.com/flthibaud/TwitchLiveNotifier/internal/utils"
)

func main() {
	// Load configuration (.env, flags, etc.)
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := utils.NewLogger(cfg)

	// Create root context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Discord client
	discordClient, err := discord.NewClient(cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to create Discord client: %v", err)
	}

	twitchServer := twitch.NewServer(cfg, logger, discordClient)

	// Start Twitch webhook server
	go func() {
		if err := twitchServer.Start(ctx); err != nil {
			logger.Errorf("Twitch server error: %v", err)
			cancel()
		}
	}()

	// Start Discord bot (blocks until shutdown)
	go func() {
		if err := discordClient.Start(ctx); err != nil {
			logger.Errorf("Discord client error: %v", err)
			cancel()
		}
	}()

	// Handle OS signals for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	logger.Info("Received shutdown signal, shutting down...")
	cancel()             // provoque la sortie de Start
	discordClient.Stop() // appelle session.Close()

	// Allow background routines to clean up
	time.Sleep(1 * time.Second)
	logger.Info("Bot has stopped")
}
