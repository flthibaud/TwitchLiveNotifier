package discord

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/flthibaud/TwitchLiveNotifier/internal/config"
	"github.com/flthibaud/TwitchLiveNotifier/internal/discord/commands"
	"github.com/flthibaud/TwitchLiveNotifier/internal/discord/events"
	"github.com/sirupsen/logrus"
)

// Client wraps the Discord session and provides start/stop functionality
type Client struct {
	session *discordgo.Session
	cfg     *config.Config
	logger  *logrus.Logger
}

// NewClient creates a new Discord client and registers event handlers
func NewClient(cfg *config.Config, logger *logrus.Logger) (*Client, error) {
	dg, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord session: %w", err)
	}

	// Set required intents
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages

	client := &Client{
		session: dg,
		cfg:     cfg,
		logger:  logger,
	}

	// Register event handlers
	dg.AddHandler(events.OnReady)
	dg.AddHandler(events.OnMessageCreate)
	dg.AddHandler(commands.PingHandler)

	return client, nil
}

// Start opens the websocket connection, registers commands, and blocks until shutdown
func (c *Client) Start(ctx context.Context) error {
	// Open Discord connection
	if err := c.session.Open(); err != nil {
		return fmt.Errorf("error opening discord session: %w", err)
	}
	c.logger.Info("Discord session opened")

	// Register slash commands now that session is open and app info is available
	commands.Register(c.session)

	// Ensure cleanup on shutdown
	defer func() {
		// Cleanup application commands
		for _, cmd := range commands.List {
			if err := c.session.ApplicationCommandDelete(c.session.State.User.ID, "", cmd.ID); err != nil {
				c.logger.Errorf("failed to delete command %s: %v", cmd.Name, err)
			}
		}
		c.session.Close()
		c.logger.Info("Discord session closed")
	}()

	// Wait for context cancellation
	<-ctx.Done()
	return nil
}

func (c *Client) Stop() {
	c.session.Close()
}

func (c *Client) SendEmbed(channelID string, embed *discordgo.MessageEmbed) error {
	if channelID == "" {
		channelID = c.cfg.NotifyChannelID
	}
	_, err := c.session.ChannelMessageSendEmbed(channelID, embed)
	return err
}
