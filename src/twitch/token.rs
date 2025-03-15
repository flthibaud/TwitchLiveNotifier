use reqwest::Client;
use serde_json::Value;
use std::time::{Duration, Instant};
use tokio::sync::Mutex;

#[allow(dead_code)]
struct TwitchToken {
    access_token: String,
    expires_at: Instant,
}

static TOKEN: Mutex<Option<TwitchToken>> = Mutex::const_new(None);

pub async fn get_twitch_access_token(client_id: &str, client_secret: &str) -> Result<String, Box<dyn std::error::Error>> {
    let mut token_guard = TOKEN.lock().await;

    if let Some(t) = &*token_guard {
        if Instant::now() < t.expires_at {
            return Ok(t.access_token.clone());
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
        access_token: new_token.clone(),
        expires_at: Instant::now() + Duration::from_secs(expires_in),
    });

    Ok(new_token)
}