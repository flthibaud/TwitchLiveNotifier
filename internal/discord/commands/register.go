package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// List holds registered slash commands
var List []*discordgo.ApplicationCommand

// Register installs all slash commands on the session
func Register(s *discordgo.Session) {
    // Collect commands
    List = []*discordgo.ApplicationCommand{PingCommand}

    // Create or update each command
    for _, cmd := range List {
        _, err := s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
        if err != nil {
          fmt.Printf("Cannot create command %s: %v\n", cmd.Name, err)
        }
    }
}