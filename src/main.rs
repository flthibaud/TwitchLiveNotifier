#![warn(clippy::str_to_string)]

mod commands;
mod twitch;
mod handler;

use poise::serenity_prelude as serenity;
use std::{
    collections::HashMap,
    env::var,
    sync::{Arc, Mutex},
    time::Duration,
};
use dotenv::dotenv;

// Types used by all command functions
type Error = Box<dyn std::error::Error + Send + Sync>;
type Context<'a> = poise::Context<'a, Data, Error>;

// Custom user data passed to all command functions
pub struct Data {
    votes: Mutex<HashMap<String, u32>>,
    discord_http: Arc<serenity::Http>,
}

async fn on_error(error: poise::FrameworkError<'_, Data, Error>) {
    // This is our custom error handler
    // They are many errors that can occur, so we only handle the ones we want to customize
    // and forward the rest to the default handler
    match error {
        poise::FrameworkError::Setup { error, .. } => panic!("Failed to start bot: {:?}", error),
        poise::FrameworkError::Command { error, ctx, .. } => {
            println!("Error in command `{}`: {:?}", ctx.command().name, error,);
        }
        error => {
            if let Err(e) = poise::builtins::on_error(error).await {
                println!("Error while handling error: {}", e)
            }
        }
    }
}

#[tokio::main]
async fn main() {
    dotenv().ok();
    tracing_subscriber::fmt::init();

    // FrameworkOptions contains all of poise's configuration option in one struct
    // Every option can be omitted to use its default value
    let options = poise::FrameworkOptions {
        commands: vec![commands::help(), commands::vote(), commands::getvotes()],
        prefix_options: poise::PrefixFrameworkOptions {
            prefix: Some("/".into()),
            edit_tracker: Some(Arc::new(poise::EditTracker::for_timespan(
                Duration::from_secs(3600),
            ))),
            additional_prefixes: vec![
                poise::Prefix::Literal("hey bot,"),
                poise::Prefix::Literal("hey bot"),
            ],
            ..Default::default()
        },
        // The global error handler for all error cases that may occur
        on_error: |error| Box::pin(on_error(error)),
        // This code is run before every command
        pre_command: |ctx| {
            Box::pin(async move {
                println!("Executing command {}...", ctx.command().qualified_name);
            })
        },
        // This code is run after a command if it was successful (returned Ok)
        post_command: |ctx| {
            Box::pin(async move {
                println!("Executed command {}!", ctx.command().qualified_name);
            })
        },
        // Every command invocation must pass this check to continue execution
        command_check: Some(|ctx| {
            Box::pin(async move {
                if ctx.author().id == 123456789 {
                    return Ok(false);
                }
                Ok(true)
            })
        }),
        // Enforce command checks even for owners (enforced by default)
        // Set to true to bypass checks, which is useful for testing
        skip_checks_for_owners: false,
        event_handler: |_ctx, event, _framework, _data| {
            Box::pin(async move {
                println!(
                    "Got an event in event handler: {:?}",
                    event.snake_case_name()
                );
                Ok(())
            })
        },
        ..Default::default()
    };

    let framework = poise::Framework::builder()
        .setup(move |ctx, _ready, framework| {
            Box::pin(async move {
                println!("Logged in as {}", _ready.user.name);
                poise::builtins::register_globally(ctx, &framework.options().commands).await?;
                let http = ctx.http.clone();
                Ok(Data {
                    votes: Mutex::new(HashMap::new()),
                    discord_http: http,
                })
            })
        })
        .options(options)
        .build();

    let token = var("DISCORD_TOKEN")
        .expect("Missing `DISCORD_TOKEN` env var, see README for more information.");
    let client_id = var("TWITCH_CLIENT_ID")
        .expect("Missing `TWITCH_CLIENT_ID` env var, see README for more information.");
    let client_secret = var("TWITCH_CLIENT_SECRET")
        .expect("Missing `TWITCH_CLIENT_SECRET` env var, see README for more information.");
    let callback_url = var("CALLBACK_URL")
        .expect("Missing `CALLBACK_URL` env var, see README for more information.");
    let intents =
        serenity::GatewayIntents::non_privileged() | serenity::GatewayIntents::MESSAGE_CONTENT;

    let client = serenity::ClientBuilder::new(token, intents)
        .framework(framework)
        .await;

    let web_port = 8080;

    let data = Arc::new(Data {
        votes: Mutex::new(HashMap::new()),
        discord_http: client.as_ref().unwrap().http.clone(),
    });

    // Démarrer le serveur web en parallèle
    let web_server = tokio::spawn(handler::setup_web_server(web_port, data));
    println!("Serveur Web lancé sur le port {}", web_port);

    if let Err(e) = twitch::create_twitch_subscription(&client_id, &client_secret, &callback_url).await {
        eprintln!("Erreur : {:?}", e);
    }

    // Start the Discord client
    let mut discord_client = client.expect("Failed to start Discord client");

    // Utilisation de tokio::select! pour exécuter les deux en parallèle
    tokio::select! {
        discord_result = discord_client.start() => {
            if let Err(e) = discord_result {
                eprintln!("Erreur du bot Discord : {:?}", e);
            }
        }
        web_result = web_server => {
            if let Err(e) = web_result {
                eprintln!("Erreur du serveur Actix : {:?}", e);
            }
        }
    }
}