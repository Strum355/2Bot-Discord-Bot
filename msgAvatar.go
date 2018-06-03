package main

import (
	"github.com/bwmarrin/discordgo"
)

func init() {
	newCommand("avatar", 0, false, false, msgAvatar).setHelp("Args: [@user]\n\nReturns the given users avatar.\nIf no user ID is given, your own avatar is sent.\n\nExample:\n`!owo avatar @Strum355#2298`").add()
}

func msgAvatar(s *discordgo.Session, m *discordgo.MessageCreate, msglist []string) {
	if len(msglist) == 1 {
		getAvatar(m.Author.ID, m, s)
		return
	}

	if len(m.Mentions) != 0 {
		getAvatar(m.Mentions[0].ID, m, s)
		return
	}

	s.ChannelMessageSend(m.ChannelID, "User not found :(")
}

func getAvatar(userID string, m *discordgo.MessageCreate, s *discordgo.Session) {
	/* 	guild, err := guildDetails(m.ChannelID, "", s)
	   	if err != nil {
	   		s.ChannelMessageSend(m.ChannelID, "There was an error finding the user :( Please try again")
	   		return
	   	} */

	// slow warmup or fast but limited to guild :( shame
	// or maybe not??? will monitor
	user, err := userDetails(userID, s)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "There was an error finding the user :( Please try again")
		return
	}

	s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
		Description: user.Username + "'s Avatar",

		Color: 0x000000,

		Image: &discordgo.MessageEmbedImage{
			URL: user.AvatarURL("2048"),
		},
	})
}
