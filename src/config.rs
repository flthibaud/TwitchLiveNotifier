use std::sync::Arc;
use poise::serenity_prelude as serenity;

pub struct Config {
  pub channel_secret: String,
  pub broadcaster_id: String,
  pub channel_id: serenity::ChannelId,
  pub client_id: String,
  pub client_secret: String,
  pub callback_url: String,
  pub discord_http: Arc<serenity::Http>,
}