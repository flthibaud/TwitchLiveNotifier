# Discord Bot RS

This project is a Discord bot written in Rust. It uses the `serenity` library to interact with the Discord API.

## Features

- [x] Respond to `!ping` command
- [x] Send a message when a streamer goes live on Twitch

## Prerequisites

- Rust (stable version)
- A Discord bot token

## Installation

1. Clone the repository:
  ```sh
  git clone https://github.com/your-username/TwitchLiveNotifier.git
  cd TwitchLiveNotifier
  ```

2. Set up your Discord bot token in a `.env` file:
  ```sh
  echo DISCORD_TOKEN=your_token_here > .env
  ```

3. Compile and run the bot:
  ```sh
  cargo run
  ```

## Usage

Once the bot is running, you can invite it to your Discord server and use the commands defined in the code.

## Contributing

Contributions are welcome! Please open an issue or a pull request to discuss the changes you want to make.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.