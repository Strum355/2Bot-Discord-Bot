package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func messageCreateEvent(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	guildDetails, err := guildDetails(m.ChannelID, "", s)
	if err != nil {
		return
	}

	var prefix string
	if val, ok := sMap.Server[guildDetails.ID]; ok {
		if prefix = val.Prefix; prefix == "" {
			prefix = c.Prefix
		}
	}

	if !strings.HasPrefix(m.Content, c.Prefix) && !strings.HasPrefix(m.Content, prefix) {
		return
	}

	parseCommand(s, m, guildDetails, func() string {
		if strings.HasPrefix(m.Content, c.Prefix) {
			return strings.TrimPrefix(m.Content, c.Prefix)
		}
		return strings.TrimPrefix(m.Content, prefix)
	}())
}

func readyEvent(s *discordgo.Session, m *discordgo.Ready) {
	log.Trace("received ready event")
	/* s.ChannelMessageSendEmbed(logChan, &discordgo.MessageEmbed{
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Info:", Value: "Received ready payload"},
		},
	}) */
	setBotGame(s)
}

func guildJoinEvent(s *discordgo.Session, m *discordgo.GuildCreate) {
	if m.Unavailable {
		log.Info("joined unavailable guild", m.Guild.ID)
		s.ChannelMessageSendEmbed(logChan, &discordgo.MessageEmbed{
			Fields: []*discordgo.MessageEmbedField{
				{"Info", "Joined unavailable guild", true},
			},
			Color: 0x00ff00,
		})
		return
	}

	user, err := userDetails(m.Guild.OwnerID, s)
	if err != nil {
		user = &discordgo.User{
			Username:      "error",
			Discriminator: "error",
		}
	}

	embed := &discordgo.MessageEmbed{
		Image: &discordgo.MessageEmbedImage{
			URL: discordgo.EndpointGuildIcon(m.Guild.ID, m.Guild.Icon),
		},

		Footer: footer,

		Fields: []*discordgo.MessageEmbedField{
			{"Name:", m.Guild.Name, true},
			{"User Count:", strconv.Itoa(m.Guild.MemberCount), true},
			{"Region:", m.Guild.Region, true},
			{"Channel Count:", strconv.Itoa(len(m.Guild.Channels)), true},
			{"ID:", m.Guild.ID, true},
			{"Owner:", user.Username + "#" + user.Discriminator, true},
		},
	}

	if _, ok := sMap.Server[m.Guild.ID]; !ok {
		//if newly joined
		embed.Color = 0x00ff00
		s.ChannelMessageSendEmbed(logChan, embed)
		log.Info("joined server", m.Guild.ID, m.Guild.Name)

		sMap.Server[m.Guild.ID] = &server{
			LogChannel:  m.Guild.ID,
			Log:         false,
			Nsfw:        false,
			JoinMessage: [3]string{"false", "", ""},
		}
	} else if val := sMap.Server[m.Guild.ID]; val.Kicked {
		//If previously kicked and then readded
		embed.Color = 0xff9a00
		s.ChannelMessageSendEmbed(logChan, embed)
		log.Info("rejoined server", m.Guild.ID, m.Guild.Name)
		sMap.Server[m.Guild.ID].Kicked = false
	}

	saveServers()
}

func guildKickedEvent(s *discordgo.Session, m *discordgo.GuildDelete) {
	if m.Unavailable {
		guild, err := guildDetails("", m.Guild.ID, s)
		if err != nil {
			log.Trace("unavailable guild", m.Guild.ID)
			return
		}
		log.Trace("unavailable guild", m.Guild.ID, guild.Name)
		return
	}

	s.ChannelMessageSendEmbed(logChan, &discordgo.MessageEmbed{
		Color:  0xff0000,
		Footer: footer,
		Fields: []*discordgo.MessageEmbedField{
			{"Name:", m.Name, true},
			{"ID:", m.Guild.ID, true},
		},
	})

	log.Info("kicked from", m.Guild.ID, m.Name)

	if guild, ok := sMap.Server[m.Guild.ID]; ok {
		guild.Kicked = true
	}

	saveServers()
}

func presenceChangeEvent(s *discordgo.Session, m *discordgo.PresenceUpdate) {
	guild, ok := sMap.Server[m.GuildID]
	if !ok || guild.Kicked || !guild.Log {
		return
	}

	memberStruct, err := memberDetails(m.GuildID, m.User.ID, s)
	if err != nil {
		return
	}

	s.ChannelMessageSend(guild.LogChannel, fmt.Sprintf("`%s is now %s`", memberStruct.User, status[m.Status]))
}

func memberJoinEvent(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	guild, ok := sMap.Server[m.GuildID]
	if !ok || guild.Kicked || len(guild.JoinMessage) != 3 {
		return
	}

	isBool, err := strconv.ParseBool(guild.JoinMessage[0])
	if err != nil {
		log.Error("couldnt parse bool", err)
		return
	}

	if !isBool || guild.JoinMessage[1] == "" {
		return
	}

	membStruct, err := userDetails(m.User.ID, s)
	if err != nil {
		return
	}

	s.ChannelMessageSend(guild.JoinMessage[2], strings.Replace(guild.JoinMessage[1], "%s", membStruct.Mention(), -1))
}
