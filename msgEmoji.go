package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var (
	errNotEmoji = errors.New("not an emoji")
)

func init() {
	newCommand("bigMoji", 0, false, false, msgEmoji).setHelp("Args: [emoji]\n\nSends a large image of the given emoji.\n" +
		"Command 'bigMoji' can be excluded for shorthand.\n\nExample:\n`!owo :smile:`\nor\n`!owo bigMoji :smile:`").add()
}

// Thanks to iopred
func emojiFile(s string) string {
	var found, filename string

	for _, r := range s {
		if filename != "" {
			filename = fmt.Sprintf("%s-%x", filename, r)
		} else {
			filename = fmt.Sprintf("%x", r)
		}

		if _, err := os.Stat(fmt.Sprintf("emoji/%s.png", filename)); err == nil {
			found = filename
		} else if found != "" {
			return found
		}
	}
	return found
}

func sendEmojiFromFile(s *discordgo.Session, m *discordgo.MessageCreate, e string) (file io.ReadCloser, err error) {
	emoji := emojiFile(e)
	if emoji == "" {
		return nil, errNotEmoji
	}

	return os.Open(fmt.Sprintf("emoji/%s.png", emoji))
}

func msgEmoji(s *discordgo.Session, m *discordgo.MessageCreate, msglist []string) {
	if len(msglist) < 1 {
		return
	}

	var emoji string
	var emojiReader io.ReadCloser
	var err error

	filename := "emoji"

	if strings.ToLower(msglist[0]) == "bigmoji" {
		if len(msglist) < 2 {
			return
		}
		emoji = msglist[1]
	} else {
		emoji = msglist[0]
	}

	submatch := emojiRegex.FindStringSubmatch(emoji)

	if len(submatch) == 0 {
		filename += ".png"
		emojiReader, err = sendEmojiFromFile(s, m, emoji)
		if err != nil {
			if err != errNotEmoji {
				log.Error("error getting emoji from file", err)
			}
			goto errored
		}
	} else {
		var url string

		switch submatch[1] {
		case "":
			url = fmt.Sprintf("https://cdn.discordapp.com/emojis/%s.png", submatch[2])
			filename += ".png"
		case "a":
			url = fmt.Sprintf("https://cdn.discordapp.com/emojis/%s.gif", submatch[2])
			filename += ".gif"
		}

		resp, err := http.Get(url)
		if err != nil {
			log.Error("error getting emoji from URL", err)
			goto errored
		}

		emojiReader = resp.Body
	}
	defer emojiReader.Close()

errored:
	if err != nil {
		if err == errNotEmoji {
			return
		}
		s.ChannelMessageSend(m.ChannelID, "There was an error getting the emoji :(")
		return
	}

	s.ChannelFileSend(m.ChannelID, filename, emojiReader)
	deleteMessage(m.Message, s)
}
