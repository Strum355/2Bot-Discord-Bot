package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	happyEmoji string = "https://cdn.discordapp.com/emojis/332968429210435585.png"
	thinkEmoji string = "https://cdn.discordapp.com/emojis/333694872802426880.png"
	reviewChan string = "334092230845267988"
	noah       string = "149612775587446784"
	logChan    string = "312352242504040448"
)

var (
	c             = &config{}
	u             = &users{}
	q             = &imageQueue{}
	lastReboot    string
	token         string
	emojiRegex    = regexp.MustCompile("<:.*?:(.*?)>")
	userIDRegex   = regexp.MustCompile("<@!?([0-9]*)>")
	fileNameRegex = regexp.MustCompile(`/`)
	status        = map[discordgo.Status]string{"dnd": "busy", "online": "online", "idle": "idle", "offline": "offline"}
	//Discord Bots, cool kidz only, social experiment, discord go
	blacklist = []string{"110373943822540800", "272873324705742848", "244133074328092673", "118456055842734083"}
)

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()

	timeNow := time.Now()
	lastReboot = timeNow.Format(time.RFC1123)[:22]
}

func main() {
	err := loadConfig()
	if err != nil {
		fmt.Println("Error loading config")
		return
	}

	loadUsers()
	loadQueue()

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session,", err)
		return
	}

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening connection,", err)
		return
	}
	defer dg.Close()

	//Register handlers
	dg.AddHandler(messageCreate)
	dg.AddHandler(joined)
	dg.AddHandler(membPresChange)
	dg.AddHandler(kicked)
	dg.AddHandler(membJoin)

	loadCommands()
	setInitialGame(dg)

	go postServerCount()
	go setQueuedImageHandlers(dg)

	log(false, "\n", `/*********BOT RESTARTED*********\`)

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func postServerCount() {
	for {
		sCount := activeServerCount()
		url := "https://bots.discord.pw/api/bots/301819949683572738/stats"
		jsonStr := []byte(`{"server_count":39}`)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))

		req.Header.Set("Authorization", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySUQiOiIxNDk2MTI3NzU1ODc0NDY3ODQiLCJyYW5kIjo4NiwiaWF0IjoxNDk3Nzk5MTkyfQ.chZmx9j84Yr0k22C46ftY8f_N2xS880KeXYFNLs3Dgs")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		_, err = client.Do(req)
		if err != nil {
			log(true, "bots.discord.pw error", err.Error())
		}
		log(true, "POSTed "+strconv.Itoa(sCount)+" to bots.discord.pw")
		time.Sleep(time.Hour * 24)
	}
}

func activeServerCount() (sCount int) {
	for _, g := range c.Servers {
		if !g.Kicked {
			sCount++
		}
	}
	return
}

func setInitialGame(s *discordgo.Session) {
	err := s.UpdateStatus(0, c.Game)
	if err != nil {
		log(true, "Update status err:", err.Error())
	}
	return
}

func loadConfig() error {
	c.Servers = make(map[string]*server)
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		return err
	}

	json.Unmarshal(file, c)
	for gID, guild := range c.Servers {
		if guild.LogChannel == "" {
			guild.LogChannel = gID
			saveConfig()
		}
	}
	return nil
}

func saveConfig() {
	out, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		log(true, "Config marshall err:", err.Error())
		return
	}

	err = ioutil.WriteFile("config.json", out, 0777)
	if err != nil {
		log(true, "Save config err:", err.Error())
	}
	return
}

func loadUsers() error {
	u.User = make(map[string]*user)
	file, err := ioutil.ReadFile("users.json")
	if err != nil {
		fmt.Println(true, "Users open err", err.Error())
		return err
	}

	json.Unmarshal(file, u)
	return nil
}

func saveUsers() {
	out, err := json.MarshalIndent(u, "", "  ")
	if err != nil {
		log(true, "Users marshall err:", err.Error())
		return
	}

	err = ioutil.WriteFile("users.json", out, 0777)
	if err != nil {
		log(true, "Save user err:", err.Error())
	}
	return
}

func loadQueue() error {
	q.QueuedMsgs = make(map[string]*queuedImage)
	file, err := ioutil.ReadFile("queue.json")
	if err != nil {
		fmt.Println(true, "Queue open err", err.Error())
		return err
	}

	json.Unmarshal(file, q)
	return nil
}

func saveQueue() {
	out, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		fmt.Println(true, "Queue marshall err:", err.Error())
		return
	}

	err = ioutil.WriteFile("queue.json", out, 0777)
	if err != nil {
		log(true, "Save queue err:", err.Error())
	}
	return
}

func randRange(min, max int) int {
	rand.Seed(time.Now().Unix())
	if max == 0 {
		return 0
	}
	return rand.Intn(max-min) + min
}

func log(timed bool, s ...string) {
	var f *os.File
	var out string
	var time1 string

	f, err := os.OpenFile("err.log", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer f.Close()

	if timed {
		time1 = time.Now().Format(time.RFC822)[:15] + " "
	}

	out = time1 + strings.Join(s, " ") + "\n"

	//if nothing failed so far, we can try to write
	//to file
	if err == nil {
		_, err = f.Write([]byte(out))
		//if nothing failed, we can return
		if err == nil {
			return
		}
	}

	//if all else fails, print log to console
	fmt.Println(err.Error() + "\n" + out)
	return
}

func findIndex(s []string, f string) int {
	for i, j := range s {
		if j == f {
			return i
		}
	}
	return -1
}

func remove(s []string, i int) []string {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getCreationTime(ID string) (t time.Time, err error) {
	i, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return
	}
	timestamp := (i >> 22) + 1420070400000
	t = time.Unix(timestamp/1000, 0)
	return
}

func codeSeg(s ...string) string {
	return "`"+strings.Join(s, " ")+"`"
}

func codeBlock(s ...string) string {
	return "```"+strings.Join(s, " ")+"```"
}

func isIn(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func isInMap(a string, aMap map[string]string) bool {
	for key := range aMap {
		if a == key {
			return true
		}
	}
	return false
}

func trimSlice(s []string) (ret []string) {
	for _, i := range s {
		ret = append(ret, strings.TrimSpace(i))
	}
	return
}

//Thanks to iopred
func emojiFile(s string) string {
	found := ""
	filename := ""
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

func guildDetails(channelID string, s *discordgo.Session) (*discordgo.Guild, error) {
	channelInGuild, err := s.State.Channel(channelID)
	if err != nil {
		log(true, "channelInGuild err:", err.Error())
		return nil, err
	}
	guildDetails, err := s.State.Guild(channelInGuild.GuildID)
	if err != nil {
		log(true, "(guildDetails) guildDetails err:", err.Error())
		return nil, err
	}
	return guildDetails, nil
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	guildDetails, err := guildDetails(m.ChannelID, s)
	if err != nil {
		return
	}

	var prefix string
	if _, ok := c.Servers[guildDetails.ID]; ok {
		if prefix = c.Servers[guildDetails.ID].Prefix; prefix == "" {
			prefix = c.Prefix
		}
	}

	if strings.HasPrefix(m.Content, prefix) {
		//code seg checks if extra whitespace is between prefix and command. Not allowed, nope :}
		//would break prefixes without trailing whitespace otherwise
		var command string
 		if len([]rune(strings.TrimPrefix(m.Content, prefix))) < 1 {
			log(true, "Uh oh why did the rune cast break")
			return
		}

		//casted to rune to index, cant index strings :(
		if string([]rune(strings.TrimPrefix(m.Content, prefix))[0]) == " " {
			command += " "
		} 
		
		msgList := strings.Fields(strings.TrimPrefix(m.Content, prefix))

		if len(msgList) > 0 {
			command += msgList[0]
			parseCommand(s, m, command, msgList)
		}
	}
	return
}

// Set all handlers for queued images, in case the bot crashes 
// with images still in queue
func setQueuedImageHandlers(s *discordgo.Session) {
	for imgNum := range q.QueuedMsgs {
		imgNumInt, err := strconv.Atoi(imgNum)
		if err != nil {
			fmt.Println(err)
			continue
		}
		go fimageReview(s, q, imgNumInt)
	}
}

func joined(s *discordgo.Session, m *discordgo.GuildCreate) {
	if m.Guild.Unavailable {
		return
	}

	guildDetails, err := s.State.Guild(m.Guild.ID)
	if err != nil {
		log(true, "Join guild struct", err.Error())
	}

	user, err := s.User(guildDetails.OwnerID)
	if err != nil {
		log(true, "Joined user struct err", err.Error())
		user = &discordgo.User {
			Username: "error",
			Discriminator: "error",
		}
	}

	if _, ok := c.Servers[m.Guild.ID]; !ok {
		//if newly joined
		s.ChannelMessageSendEmbed(logChan, &discordgo.MessageEmbed{
			Color: 65280,

			Image: &discordgo.MessageEmbedImage {
				URL: discordgo.EndpointGuildIcon(m.Guild.ID, m.Guild.Icon),
			},

			Footer: &discordgo.MessageEmbedFooter{
				Text: "Brought to you by 2Bot :)\nLast Bot reboot: " + lastReboot + " GMT",
			},

			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{Name: "Name:", Value: m.Guild.Name, Inline: true},
				&discordgo.MessageEmbedField{Name: "User Count:", Value: strconv.Itoa(m.Guild.MemberCount), Inline: true},
				&discordgo.MessageEmbedField{Name: "Region:", Value: m.Guild.Region, Inline: true},
				&discordgo.MessageEmbedField{Name: "Channel Count:", Value: strconv.Itoa(len(m.Guild.Channels)), Inline: true},
				&discordgo.MessageEmbedField{Name: "ID:", Value: m.Guild.ID, Inline: true},
				&discordgo.MessageEmbedField{Name: "Owner:", Value: user.Username + "#" + user.Discriminator, Inline: true},
			},
		})

		c.Servers[m.Guild.ID] = &server{
			LogChannel:  m.Guild.ID,
			Log:         false,
			Nsfw:        false,
			JoinMessage: [3]string{"false", "", ""},
		}

		log(true, "Joined server", m.Guild.ID, m.Guild.Name)
	}else if val := c.Servers[m.Guild.ID]; val.Kicked == true {
		//If previously kicked and then readded
		s.ChannelMessageSendEmbed(logChan, &discordgo.MessageEmbed{
			Color: 16751104,

			Image: &discordgo.MessageEmbedImage {
				URL: discordgo.EndpointGuildIcon(m.Guild.ID, m.Guild.Icon),
			},

			Footer: &discordgo.MessageEmbedFooter{
				Text: "Brought to you by 2Bot :)\nLast Bot reboot: " + lastReboot + " GMT",
			},

			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{Name: "Name:", Value: m.Guild.Name, Inline: true},
				&discordgo.MessageEmbedField{Name: "User Count:", Value: strconv.Itoa(m.Guild.MemberCount), Inline: true},
				&discordgo.MessageEmbedField{Name: "Region:", Value: m.Guild.Region, Inline: true},
				&discordgo.MessageEmbedField{Name: "Channel Count:", Value: strconv.Itoa(len(m.Guild.Channels)), Inline: true},
				&discordgo.MessageEmbedField{Name: "ID:", Value: m.Guild.ID, Inline: true},
				&discordgo.MessageEmbedField{Name: "Owner:", Value: user.Username + "#" + user.Discriminator, Inline: true},
			},
		})

		log(true, "Rejoined server", m.Guild.ID, m.Guild.Name)
	}

	c.Servers[m.Guild.ID].Kicked = false
	saveConfig()

	return
}

func kicked(s *discordgo.Session, m *discordgo.GuildDelete) {
	if !m.Unavailable {
		s.ChannelMessageSendEmbed(logChan, &discordgo.MessageEmbed{
			Color: 16711680,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Brought to you by 2Bot :)\nLast Bot reboot: " + lastReboot + " GMT",
			},
			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{Name: "Name:", Value: m.Guild.Name, Inline: true},
				&discordgo.MessageEmbedField{Name: "ID:", Value: m.Guild.ID, Inline: true},
			},
		})

		log(true, "Kicked from", m.Guild.ID, m.Name)
		c.Servers[m.Guild.ID].Kicked = true
		saveConfig()
	}
	return
}

func membPresChange(s *discordgo.Session, m *discordgo.PresenceUpdate) {
	if guild, ok := c.Servers[m.GuildID]; ok && !guild.Kicked {
		if guild.Log {
			guildDetails, err := s.State.Guild(m.GuildID)
			if err != nil {
				log(true, "(membPresChange) guildDetails err:", err.Error())
				return
			}

			memberStruct, err := s.State.Member(m.GuildID, m.User.ID)
			if err != nil {
				log(true, guildDetails.Name, m.GuildID, err.Error())
				return
			}

			s.ChannelMessageSend(guild.LogChannel, fmt.Sprintf("`%s is now %s`", memberStruct.User, status[m.Status]))
		}
	}
	return
}

func membJoin(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	if guild, ok := c.Servers[m.GuildID]; ok && !guild.Kicked {
		if len(guild.JoinMessage) == 3 {
			isBool, err := strconv.ParseBool(guild.JoinMessage[0])
			if err != nil {
				log(true, "Config join msg bool err", err.Error())
				return
			}
			if isBool && guild.JoinMessage[1] != "" {
				guildDetails, err := s.State.Guild(m.GuildID)
				if err != nil {
					log(true, "(membJoin) guildDetails err:", err.Error())
					return
				}

				membStruct, err := s.User(m.User.ID)
				if err != nil {
					log(true, guildDetails.Name, m.GuildID, err.Error())
					return
				}

				message := strings.Replace(guild.JoinMessage[1], "%s", membStruct.Mention(), -1)
				s.ChannelMessageSend(guild.JoinMessage[2], message)
			}
		}
	}
	return
}
