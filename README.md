# TwitchLiveNotifier

TwitchLiveNotifier is a modular Go application for a Discord bot that notifies multiple Twitch channels when they go live using Twitch EventSub webhooks.

## Prerequisites

- **Go** 1.20 or newer ([download](https://go.dev/dl/)).
- A **Discord Bot Token** (create one in the [Discord Developer Portal](https://discord.com/developers/applications)).
- A **Twitch Application** with **Client ID** and **Client Secret** (create one in the [Twitch Developer Console](https://dev.twitch.tv/console/apps)).
- A public **HTTPS** endpoint (e.g., [ngrok](https://ngrok.com/), localtunnel, or a public server) to receive `/webhook` callbacks.

## Environment Variables

Copy `.env.example` to `.env` at the project root and fill in your values:

```dotenv
# Local bind address for webhook server
PORT=8080

# Discord settings
BOT_TOKEN=YOUR_DISCORD_BOT_TOKEN
NOTIFY_CHANNEL_ID=DISCORD_TEXT_CHANNEL_ID_FOR_NOTIFICATIONS

# Twitch EventSub settings
TWITCH_CLIENT_ID=YOUR_TWITCH_CLIENT_ID
TWITCH_CLIENT_SECRET=YOUR_TWITCH_CLIENT_SECRET
TWITCH_WEBHOOK_SECRET=YOUR_EVENTSUB_SECRET
# Comma-separated list of Twitch broadcaster user IDs
TWITCH_BROADCASTER_IDS=12345678,87654321

# Public HTTPS URL for webhook callbacks
CALLBACK_URL=https://your-app.ngrok.io

# Logging level (debug, info, warn, error)
LOG_LEVEL=info
```

## Installation

1. Clone the repository:
```bash
git clone https://github.com/flthibaud/discord-twitch-bot.git
cd discord-twitch-bot
``` 
2. Install dependencies and tidy modules:
```bash
go mod tidy
```

## Running the Bot

### Development

```bash
cp .env.example .env    # copy and edit .env
# fill in .env with your credentials
go run ./cmd/bot/main.go
```

### Production (Build Binary)

```bash
go build -o discord-twitch-bot ./cmd/bot
./discord-twitch-bot
```

## Project Structure

```
discord-twitch-bot/
├── .env.example             # Example environment variables
├── cmd/
│   └── bot/
│       └── main.go          # Entry point
├── internal/
│   ├── config/
│   │   └── config.go        # .env loading and validation
│   ├── utils/
│   │   └── logger.go        # Logrus-based logger
│   ├── discord/
│   │   ├── client.go        # Discord client wrapper (Start/Stop)
│   │   ├── commands/        # Slash command definitions and registration
│   │   └── events/          # Discord event handlers
│   └── twitch/
│       ├── webhook.go       # HTTP server and EventSub management
│       └── stream_info.go   # Twitch Helix API client for stream info
├── go.mod
└── README.md                # This file
```

## Adding New Slash Commands

1. Create a Go file in `internal/discord/commands/`.
2. Define an `ApplicationCommand` and its handler function.
3. The `commands.Register` function will automatically register all commands on bot startup.

## Adding New Event Handlers

1. Create a Go file in `internal/discord/events/`.
2. Add `dg.AddHandler(YourEventHandler)` in `client.go` to wire it up.

## Twitch EventSub Workflow

On startup, the bot:

1. Retrieves an OAuth app access token from Twitch.
2. Iterates over each `TWITCH_BROADCASTER_ID`:
   - Lists existing EventSub subscriptions.
   - Deletes outdated subscriptions if callback URL changed.
   - Creates a new `stream.online` subscription if none is valid.
3. Starts an HTTP server on `TWITCH_WEBHOOK_ADDR`, serving `/webhook`.

When a streamer goes live (`stream.online` event), the bot:

1. Verifies the HMAC signature using `TWITCH_WEBHOOK_SECRET`.
2. Parses the JSON payload for `broadcaster_user_name`, `title`, `game_name`, `viewer_count`, etc.
3. Builds and sends a rich Discord embed to `NOTIFY_CHANNEL_ID`.

## Roadmap

Planned features and improvements:

- **Database Integration**: Add support for a database (PostgreSQL, SQLite, etc.).

- **Dynamic Channel Management**: Implement bot commands (e.g., /addchannel, /removechannel) restricted to a specific Discord role for adding or removing Twitch channels at runtime.

- **Permission Controls**: Leverage Discord roles to manage who can execute administrative commands.

- **Unit & Integration Tests**: Improve coverage for core modules (Discord commands, Twitch webhook handling).

## Contributing

Feel free to fork the repo and submit pull requests for improvements or new features. Please ensure your code follows Go conventions and is well-documented.

