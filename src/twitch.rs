use reqwest::Client;
use serde_json::Value;
use std::time::{Duration, Instant};
use tokio::sync::Mutex;
use serde::{Deserialize, Serialize};

#[allow(dead_code)]
struct TwitchToken {
    token: String,
    expires_at: Instant,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Stream {
    id: String,
    user_id: String,
    user_login: String,
    user_name: String,
    game_id: String,
    game_name: String,
    #[serde(rename = "type")]
    stream_type: String,
    title: String,
    viewer_count: u32,
    started_at: String,
    language: String,
    thumbnail_url: String,
    // D'autres champs peuvent être ajoutés si besoin
}

#[derive(Serialize, Deserialize, Debug)]
struct GetStreamsResponse {
    data: Vec<Stream>,
}

static TOKEN: Mutex<Option<TwitchToken>> = Mutex::const_new(None);

pub async fn get_twitch_access_token(client_id: &str, client_secret: &str) -> Result<String, Box<dyn std::error::Error>> {
    let mut token_guard = TOKEN.lock().await;

    if let Some(t) = &*token_guard {
        if Instant::now() < t.expires_at {
            return Ok(t.token.clone());
        }
    }

    let client = Client::new();
    let url = format!(
        "https://id.twitch.tv/oauth2/token?client_id={}&client_secret={}&grant_type=client_credentials",
        client_id, client_secret
    );

    let res = client.post(&url).send().await?.json::<Value>().await?;
    let new_token = res["access_token"].as_str().unwrap().to_string();
    let expires_in = res["expires_in"].as_u64().unwrap_or(14400);

    *token_guard = Some(TwitchToken {
        token: new_token.clone(),
        expires_at: Instant::now() + Duration::from_secs(expires_in),
    });

    Ok(new_token)
}

pub async fn create_twitch_subscription(client_id: &str, client_secret: &str, callback_url: &str) -> Result<(), Box<dyn std::error::Error>> {
  let events = vec!["stream.online", "stream.offline"];

  let access_token = get_twitch_access_token(client_id, client_secret).await?;

  let client = Client::new();
  
  for event in events {
      let body = serde_json::json!({
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

/// Récupère les infos de stream pour un utilisateur donné.
/// Renvoie `Ok(Some(stream))` si l'utilisateur est en live, sinon `Ok(None)`.
pub async fn get_stream_info(
    broadcaster_id: &str,
    client_id: &str,
    access_token: &str,
) -> Result<Option<Stream>, reqwest::Error> {
    let client = Client::new();
    let url = format!("https://api.twitch.tv/helix/streams?user_id={}", broadcaster_id);

    let response = client
        .get(&url)
        .header("Client-ID", client_id)
        .header("Authorization", format!("Bearer {}", access_token))
        .send()
        .await?;

    // Désérialise la réponse JSON
    let get_streams: GetStreamsResponse = response.json().await?;
    
    // Si le vecteur data n'est pas vide, ça veut dire que l'utilisateur est live.
    Ok(get_streams.data.into_iter().next())
}