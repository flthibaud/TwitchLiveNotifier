use reqwest::Client;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Debug)]
pub struct Stream {
    pub id: String,
    pub user_id: String,
    pub user_login: String,
    pub user_name: String,
    pub game_id: String,
    pub game_name: String,
    #[serde(rename = "type")]
    pub stream_type: String,
    pub title: String,
    pub viewer_count: u32,
    pub started_at: String,
    pub language: String,
    pub thumbnail_url: String,
    // D'autres champs peuvent être ajoutés si besoin
}

#[derive(Serialize, Deserialize, Debug)]
struct GetStreamsResponse {
    data: Vec<Stream>,
}

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