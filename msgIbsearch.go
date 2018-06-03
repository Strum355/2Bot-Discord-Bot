package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func init() {
	newCommand("ibsearch", 0, false, false, msgIbsearch).setHelp("Args: [search] | rating=[e,s,q] | format=[gif,png,jpg]\n\n" +
		"Returns a random image from ibsearch for the given search term with the given filters applied.\n\n" +
		"Example:\n`!owo ibsearch lewds | rating=e | format=gif`")
}

func msgIbsearch(s *discordgo.Session, m *discordgo.MessageCreate, msglist []string) {
	if len(msglist) < 2 {
		return
	}

	channel, err := channelDetails(m.ChannelID, s)
	if err != nil {
		return
	}

	guild, err := guildDetails("", channel.GuildID, s)
	if err != nil {
		return
	}

	if !sMap.Server[guild.ID].Nsfw && !strings.Contains(channel.Name, "nsfw") && !channel.NSFW {
		s.ChannelMessageSend(m.ChannelID, "NSFW is disabled on this server~")
		return
	}

	s.ChannelTyping(m.ChannelID)

	URL, err := url.Parse("https://ibsearch.xxx")
	if err != nil {
		log.Error("IBSearch query error", err)
		return
	}

	var queries []string

	queryList := strings.Split(strings.Join(remove(msglist, 0), " "), "|")

	for i, item := range queryList {
		if strings.Contains(item, "=") {
			queries = append(queries, strings.TrimSpace(strings.Split(queryList[i], "=")[0]))
		}
	}

	var finalQuery string

	filters := []string{"rating", "format"}

	for _, item1 := range filters {
		for i, item2 := range queries {
			if strings.Contains(item1, item2) {
				finalQuery += strings.Replace(queryList[i+1], " ", "", -1) + " "
			}
		}
	}

	//Assemble the URL
	var par url.Values
	URL.Path += "/api/v1/images.json"
	par.Add("q", strings.TrimSpace(queryList[0])+" "+finalQuery+"random:")
	par.Add("limit", "1")

	//Public key that is for free, worst that can happen is that
	//i hit the ratelimit, but please dont do that to me
	par.Add("key", "2480CFA681A7A882CB33C0E4BA00A812C6F906A6")
	URL.RawQuery = par.Encode()

	client := http.Client{Timeout: time.Second * 2}
	page, err := client.Get(URL.String())
	if err != nil {
		log.Error("Ibsearch http error", err)
		s.ChannelMessageSend(m.ChannelID, "Error getting results from ibsearch")
		return
	}
	defer page.Body.Close()

	if page.StatusCode != http.StatusOK {
		s.ChannelMessageSend(m.ChannelID, "IBSearch didn't respond :(")
		return
	}

	var ibsearchStruct [1]ibStruct
	if err := json.NewDecoder(page.Body).Decode(&ibsearchStruct); err != nil {
		log.Error("IBSearch json unmarshal err", err)
		s.ChannelMessageSend(m.ChannelID, "No results ¯\\_(ツ)_/¯")
		return
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s searched for `%s` \nhttps://%s.ibsearch.xxx/%s", m.Author.Username, queryList[0], ibsearchStruct[0].Server, ibsearchStruct[0].Path))
}
