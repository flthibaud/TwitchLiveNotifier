package events

import "github.com/bwmarrin/discordgo"

// OnMessageCreate replies to "Hello" messages
func OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
    if m.Author.ID == s.State.User.ID || m.Author.Bot {
        return
    }
    if m.Content == "Hello" {
        reply := "Hello, " + m.Author.Username + "!"
        s.ChannelMessageSend(m.ChannelID, reply)
    }
}

// OnReady sets the initial status when the bot is ready
func OnReady(s *discordgo.Session, r *discordgo.Ready) {
    s.UpdateCustomStatus("Watching Twitch")
}