package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gobwas/glob"

	"github.com/Knetic/govaluate"

	"github.com/bwmarrin/discordgo"
)

func init() {
	newCommand("purge",
		discordgo.PermissionAdministrator|discordgo.PermissionManageMessages|discordgo.PermissionManageServer,
		true, msgPurgeEx).setHelp("purge messages using an expression, ex `purge 100 glob('*necroforger*', username) || glob('*owo*', content)` will purge all messages either from necroforger or containing the substring owo").add()
}

// msgPurgeEx is an extra purge function that uses govaluate
func msgPurgeEx(s *discordgo.Session, m *discordgo.MessageCreate, msglist []string) {
	if len(msglist) < 2 {
		s.ChannelMessageSend(m.ChannelID, "This command requires an expression to delete messages with")
		return
	}

	var limit int
	if n, err := strconv.Atoi(msglist[1]); err == nil {
		limit = n
	} else {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s is not a valid number, please enter an integer", msglist[1]))
		return
	}

	if len(msglist) > 2 {
		exp := strings.Join(msglist[2:], " ")
		goexp, err := govaluate.NewEvaluableExpressionWithFunctions(exp, map[string]govaluate.ExpressionFunction{
			// First argument glob expression, second is comparison
			"glob": func(arguments ...interface{}) (interface{}, error) {
				if len(arguments) < 2 {
					return nil, errors.New("Glob requires two arguments")
				}

				var e glob.Glob
				var content string
				var err error

				// Glob pattern
				if s, ok := arguments[0].(string); ok {
					e, err = glob.Compile(s)
					if err != nil {
						return nil, err
					}
				} else {
					return nil, errors.New("first argument to glob must be a string")
				}

				// Comparison content
				if s, ok := arguments[1].(string); ok {
					content = s
				} else {
					return nil, errors.New("Second argument to glob must be a string")
				}

				return e.Match(content), nil
			},
		})
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Something went wrong with constructing your expression: "+exp+" "+err.Error())
			return
		}

		err = messagePurge(limit, func(msg *discordgo.Message) (bool, error) {
			result, err := goexp.Evaluate(map[string]interface{}{
				"msg":               msg,
				"username":          msg.Author.Username,
				"userid":            msg.Author.ID,
				"content":           msg.Content,
				"id":                msg.ID,
				"mentions_everyone": msg.MentionEveryone,
				"channelid":         msg.ChannelID,
			})
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "error evaluating expression: "+err.Error())
				return false, err
			}

			b, ok := result.(bool)
			if !ok {
				s.ChannelMessageSend(m.ChannelID, "Expression result was not a boolean value")
				return false, err
			}

			return b, nil
		}, s, m)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error purging messages: "+err.Error())
		}

	} else {
		standardPurge(limit, s, m)
	}
}

func msgPurge(s *discordgo.Session, m *discordgo.MessageCreate, msglist []string) {
	if len(msglist) < 2 {
		s.ChannelMessageSend(m.ChannelID, "Gotta specify a number of messages to delete~")
		return
	}

	purgeAmount, err := strconv.Atoi(msglist[1])
	if err != nil {
		if strings.HasPrefix(msglist[1], "@") {
			msglist[1] = "@" + zerowidth + strings.TrimPrefix(msglist[1], "@")
		}
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("How do i delete %s messages? Please only give numbers!", msglist[1]))
		return
	}

	var userToPurge string
	if len(msglist) == 3 {
		submatch := userIDRegex.FindStringSubmatch(msglist[2])
		if len(submatch) == 0 {
			s.ChannelMessageSend(m.ChannelID, "Couldn't find that user :(")
			return
		}
		userToPurge = submatch[1]
	}

	deleteMessage(m.Message, s)

	if userToPurge == "" {
		err = standardPurge(purgeAmount, s, m)
	} else {
		err = userPurge(purgeAmount, s, m, userToPurge)
	}

	if err == nil {
		msg, _ := s.ChannelMessageSend(m.ChannelID, "Successfully deleted :ok_hand:")
		time.Sleep(time.Second * 5)
		deleteMessage(msg, s)
	}
}

func getMessages(amount int, id string, s *discordgo.Session) (list []*discordgo.Message, err error) {
	list, err = s.ChannelMessages(id, amount, "", "", "")
	if err != nil {
		log.Error("error getting messages to delete", err)
	}
	return
}

func messagePurge(purgeAmount int, purgefn func(*discordgo.Message) (bool, error), s *discordgo.Session, m *discordgo.MessageCreate) error {
	var limit = purgeAmount
	var beforeID string
	messages := []string{}

	// Obtain messages
	for limit > 0 {
		var l int

		// Collect messages in batches of 100 at a time
		if limit > 100 {
			l = 100
		} else {
			l = limit
		}

		msgs, err := s.ChannelMessages(m.ChannelID, l, beforeID, "", "")
		if err != nil {
			return err
		}

		// Done collecting messages, probably reached the end of the channel
		if len(msgs) == 0 {
			break
		}

		// Collect messages before this ID in the next round
		beforeID = msgs[len(msgs)-1].ID

		// Count the number of messages that pass the purge function
		// To subtract them from the limit
		var count int
		for _, v := range msgs {
			b, err := purgefn(v)
			if err != nil {
				return err
			}
			if b {
				count++
				messages = append(messages, v.ID)
			}
		}

		limit -= count
	}

	return s.ChannelMessagesBulkDelete(m.ChannelID, messages)
}

func standardPurge(purgeAmount int, s *discordgo.Session, m *discordgo.MessageCreate) error {
	return messagePurge(purgeAmount, func(*discordgo.Message) (bool, error) {
		return true, nil
	}, s, m)
}

func userPurge(purgeAmount int, s *discordgo.Session, m *discordgo.MessageCreate, userToPurge string) error {
	var outOfDate bool
	for purgeAmount > 0 {
		del := purgeAmount % 100
		var purgeList []string

		for len(purgeList) < del {
			list, err := getMessages(100, m.ChannelID, s)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "There was an issue deleting messages :(")
				return err
			}

			//if more was requested to be deleted than exists
			if len(list) == 0 {
				break
			}

			for _, msg := range list {
				if len(purgeList) == del {
					break
				}

				if msg.Author.ID != userToPurge {
					continue
				}

				timeSince, err := getMessageAge(msg, s, m)
				if err != nil {
					//if the time is malformed for whatever reason, we'll try the next message
					continue
				}

				if timeSince.Hours()/24 >= 14 {
					outOfDate = true
					break
				}

				purgeList = append(purgeList, msg.ID)
			}

			if outOfDate {
				break
			}
		}

		if err := massDelete(purgeList, s, m); err != nil {
			return err
		}

		if outOfDate {
			break
		}

		purgeAmount -= len(purgeList)
	}

	return nil
}

func massDelete(list []string, s *discordgo.Session, m *discordgo.MessageCreate) (err error) {
	if err = s.ChannelMessagesBulkDelete(m.ChannelID, list); err != nil {
		s.ChannelMessageSend(m.ChannelID, "There was an issue deleting messages :(")
		log.Error("error purging", err)
	}
	return
}

func getMessageAge(msg *discordgo.Message, s *discordgo.Session, m *discordgo.MessageCreate) (time.Duration, error) {
	then, err := msg.Timestamp.Parse()
	if err != nil {
		log.Error("error parsing time", err)
		return time.Duration(0), err
	}
	return time.Since(then), nil
}
