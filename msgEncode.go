package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	newCommand("encode", 0, false, false, msgEncode).setHelp("Args: [base] [text]\n\nBases: `base64`, `bcrypt`, `md5`, `sh256`\nEncodes the given text in the given base.\n\nExample:\n`!owo encode md5 some text`").add()
}

func msgEncode(s *discordgo.Session, m *discordgo.MessageCreate, msglist []string) {
	if len(msglist) < 3 {
		return
	}

	base := strings.ToLower(msglist[1])
	text := strings.Join(msglist[2:], " ")
	var output []byte

	s.ChannelTyping(m.ChannelID)

	switch base {
	case "base64":
		output = []byte(base64.StdEncoding.EncodeToString([]byte(text)))
	case "bcrypt":
		var err error
		if output, err = bcrypt.GenerateFromPassword([]byte(text), 14); err != nil {
			log.Error("bcrypt err", err)
			return
		}
	case "md5":
		hash := md5.Sum([]byte(text))
		output = make([]byte, hex.EncodedLen(len(hash)))
		hex.Encode(output, hash[:])
	case "sha256":
		hash := sha256.Sum256([]byte(text))
		output = make([]byte, hex.EncodedLen(len(hash)))
		hex.Encode(output, hash[:])
	default:
		s.ChannelMessageSend(m.ChannelID, "Base not supported")
		return
	}

	s.ChannelMessageSend(m.ChannelID, string(output))
}
