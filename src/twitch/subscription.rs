use reqwest::Client;
use serde_json::json;

use crate::twitch::token::get_twitch_access_token;

pub async fn create_twitch_subscription(client_id: &str, client_secret: &str, callback_url: &str) -> Result<(), Box<dyn std::error::Error>> {
    let events = vec!["stream.online", "stream.offline"];

    let access_token = get_twitch_access_token(client_id, client_secret).await?;

    let client = Client::new();
    
    for event in events {
        let body = json!({
            "type": event,
            "version": "1",
            "condition": {
                "broadcaster_user_id": "1279343673" // Remplace avec l'ID du streamer
            },
            "transport": {
                "method": "webhook",
                "callback": callback_url, // URL où Twitch enverra les notifications
                "secret": "ton_channel_secret"
            }
        });

        let res = client
            .post("https://api.twitch.tv/helix/eventsub/subscriptions")
            .header("Authorization", format!("Bearer {}", access_token))
            .header("Client-Id", client_id)
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