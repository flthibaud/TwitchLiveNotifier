use std::sync::Arc;

use hmac::{Hmac, Mac};
use serde_json::Value;
use sha2::Sha256;
use subtle::ConstantTimeEq;
use warp::http::{HeaderMap, StatusCode};
use warp::Filter;

use crate::config::Config;
use crate::twitch;
use crate::twitch::token::get_twitch_access_token;

type HmacSha256 = Hmac<Sha256>;

// Nos erreurs personnalisées pour Warp
#[derive(Debug)]
struct SignatureError;
impl warp::reject::Reject for SignatureError {}

#[derive(Debug)]
struct JsonError;
impl warp::reject::Reject for JsonError {}

#[derive(Debug)]
struct UnhandledEvent(String);
impl warp::reject::Reject for UnhandledEvent {}

fn validate_signature(headers: &HeaderMap, body: &str, channel_secret: &str) -> bool {
    // Récupération des headers nécessaires
    let signature = headers
        .get("Twitch-Eventsub-Message-Signature")
        .and_then(|s| s.to_str().ok())
        .unwrap_or("");
    let message_id = headers
        .get("Twitch-Eventsub-Message-Id")
        .and_then(|s| s.to_str().ok())
        .unwrap_or("");
    let timestamp = headers
        .get("Twitch-Eventsub-Message-Timestamp")
        .and_then(|s| s.to_str().ok())
        .unwrap_or("");

    if signature.is_empty() || message_id.is_empty() || timestamp.is_empty() {
        return false;
    }

    // Construction du message en concaténant message_id, timestamp et le body
    let message = format!("{}{}{}", message_id, timestamp, body);

    // Calcul du HMAC-SHA256 avec la clé secrète
    let mut mac = HmacSha256::new_from_slice(channel_secret.as_bytes())
        .expect("HMAC peut accepter n'importe quelle taille de clé !");
    mac.update(message.as_bytes());
    let result = mac.finalize().into_bytes();

    // Formatage de la signature attendue, qui doit commencer par "sha256="
    let expected_signature = format!("sha256={:x}", result);

    // Comparaison en temps constant pour éviter les attaques side-channel
    expected_signature
        .as_bytes()
        .ct_eq(signature.as_bytes())
        .into()
}

fn health_check() -> impl Filter<Extract = impl warp::Reply, Error = warp::Rejection> + Clone {
    warp::path("ping").and(warp::get()).map(|| "pong")
}

async fn handle_webhook(
    headers: HeaderMap,
    body_bytes: bytes::Bytes,
    config: Arc<Config>,
) -> Result<impl warp::Reply, warp::Rejection> {
    // Convertit le body en String
    let body = String::from_utf8(body_bytes.to_vec()).unwrap_or_default();

    // Validation de la signature, sinon on rejette la requête
    if !validate_signature(&headers, &body, &config.channel_secret) {
        return Err(warp::reject::custom(SignatureError));
    }

    // Récupère le type de message
    let message_type = headers
        .get("Twitch-Eventsub-Message-Type")
        .and_then(|val| val.to_str().ok())
        .unwrap_or("");

    // Si c'est une vérification de webhook, renvoie le challenge
    if message_type == "webhook_callback_verification" {
        let json: Value =
            serde_json::from_str(&body).map_err(|_| warp::reject::custom(JsonError))?;
        if let Some(challenge) = json.get("challenge").and_then(|c| c.as_str()) {
            return Ok(warp::reply::with_status(
                challenge.to_string(),
                StatusCode::OK,
            ));
        } else {
            return Err(warp::reject::custom(JsonError));
        }
    }

    // Pour les autres événements, on parse le JSON et on regarde le type d'abonnement
    let json: Value = serde_json::from_str(&body).map_err(|_| warp::reject::custom(JsonError))?;
    if let Some(subscription) = json.get("subscription") {
        if let Some(event_type) = subscription.get("type").and_then(|t| t.as_str()) {
            match event_type {
                "stream.online" => {
                    println!("Stream en ligne : mise à jour de la date live.");

                    let access_token = match get_twitch_access_token(&config.client_id, &config.client_secret).await {
                        Ok(token) => token,
                        Err(e) => {
                            eprintln!("Erreur lors de la récupération du token d'accès Twitch : {:?}", e);
                            return Ok(warp::reply::with_status(
                                "".to_string(),
                                StatusCode::INTERNAL_SERVER_ERROR,
                            ));
                        }
                    };

                    // Récupération des informations du stream
                    match twitch::stream_info::get_stream_info(
                        &config.broadcaster_id,
                        &config.client_id,
                        &access_token,
                    )
                    .await
                    {
                        Ok(Some(stream)) => {
                            let message = format!(
                                "Le streamer {} est en live ! Titre: {}, Jeu: {}, Viewers: {}",
                                stream.user_name,
                                stream.title,
                                stream.game_name,
                                stream.viewer_count
                            );

                            // Envoi d'un message embed sur Discord
                            if let Err(e) = config.channel_id.say(&config.discord_http, &message).await {
                                eprintln!("Erreur lors de l'envoi du message Discord : {:?}", e);
                            }
                        }
                        Ok(None) => {
                            eprintln!("Le streamer n'est pas en live.");
                        }
                        Err(e) => {
                            eprintln!(
                                "Erreur lors de la récupération des informations du stream : {:?}",
                                e
                            );
                        }
                    }

                    return Ok(warp::reply::with_status(
                        "".to_string(),
                        StatusCode::NO_CONTENT,
                    ));
                }
                "stream.offline" => {
                    println!("Stream hors ligne : réinitialisation de la date live.");
                    return Ok(warp::reply::with_status(
                        "".to_string(),
                        StatusCode::NO_CONTENT,
                    ));
                }
                _ => {
                    let sub_type = subscription
                        .get("type")
                        .and_then(|t| t.as_str())
                        .unwrap_or("inconnu");
                    return Err(warp::reject::custom(UnhandledEvent(sub_type.to_string())));
                }
            }
        }
    }
    Err(warp::reject::custom(JsonError))
}

pub async fn setup_web_server(port: u16, config: Arc<Config>) {
    let config_filter = warp::any().map(move || config.clone());

    // On initialise nos routes via des fonctions dédiées
    let health_route = health_check();
    let webhook_route = warp::path!("twitch" / "webhook")
        .and(warp::post())
        // On récupère tous les headers
        .and(warp::header::headers_cloned())
        // On récupère le body en tant que bytes
        .and(warp::body::bytes())
        .and(config_filter)
        .and_then(handle_webhook);

    // On combine les routes avec l'opérateur `or`
    let routes = health_route.or(webhook_route);

    warp::serve(routes).run(([0, 0, 0, 0], port)).await;
}
