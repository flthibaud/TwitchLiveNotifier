use crate::{Context, Error};

/// Show this help menu
#[poise::command(prefix_command, track_edits, slash_command)]
pub async fn help(
    ctx: Context<'_>,
    #[description = "Specific command to show help about"]
    #[autocomplete = "poise::builtins::autocomplete_command"]
    command: Option<String>,
) -> Result<(), Error> {
    poise::builtins::help(
        ctx,
        command.as_deref(),
        poise::builtins::HelpConfiguration {
            extra_text_at_bottom: "This is an example bot made to showcase features of my custom Discord bot framework",
            ..Default::default()
        },
    )
    .await?;
    Ok(())
}

/// Respond to command ping with "Pong!"
///
/// This command is used to check if the bot is online
#[poise::command(prefix_command, slash_command)]
pub async fn ping(
    ctx: Context<'_>,
) -> Result<(), Error> {
    ctx.say("Pong!").await?;
    Ok(())
}