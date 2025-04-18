package commands

import (
	"github.com/bwmarrin/discordgo"
)

// PingCommand defines the /ping command
var PingCommand = &discordgo.ApplicationCommand{
    Name:        "ping",
    Description: "Répond pong",
}

// PingHandler responds to /ping with "Pong!"
func PingHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
    if i.ApplicationCommandData().Name != "ping" {
        return
    }
    s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: "Pong!",
        },
    })
}