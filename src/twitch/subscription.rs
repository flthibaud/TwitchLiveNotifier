use reqwest::Client;
use serde_json::json;
use std::sync::Arc;

use crate::config::Config;
use crate::twitch::token::get_twitch_access_token;

pub async fn create_twitch_subscription(config: Arc<Config>) -> Result<(), Box<dyn std::error::Error>> {
    let events = vec!["stream.online", "stream.offline"];

    let access_token = get_twitch_access_token(&config.client_id, &config.client_secret).await?;

    let client = Client::new();
    
    for event in events {
        let body = json!({
            "type": event,
            "version": "1",
            "condition": {
                "broadcaster_user_id": &config.broadcaster_id,
            },
            "transport": {
                "method": "webhook",
                "callback": &config.callback_url, // URL où Twitch enverra les notifications
                "secret": &config.channel_secret,
            }
        });

        let res = client
            .post("https://api.twitch.tv/helix/eventsub/subscriptions")
            .header("Authorization", format!("Bearer {}", access_token))
            .header("Client-Id", &config.client_id)
            .header("Content-Type", "application/json")
            .json(&body)
            .send()
            .await?;

        if res.status().is_success() {
            println!("✅ Souscription à `{}` réussie !", event);
        } else {
            println!("❌ Erreur lors de la souscription `{}` : {:?}", event, res.text().await?);
        }
    }

    Ok(())
}