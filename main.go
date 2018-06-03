/*
	Copyright (C) 2017  Noah Santschi-Cooney

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU Affero General Public License as published
    by the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Affero General Public License for more details.

    You should have received a copy of the GNU Affero General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"bytes"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi"
)

const (
	happyEmoji string = "https://cdn.discordapp.com/emojis/332968429210435585.png"
	thinkEmoji string = "https://cdn.discordapp.com/emojis/333694872802426880.png"
	reviewChan string = "334092230845267988"
	noah       string = "149612775587446784"
	logChan    string = "312352242504040448"
	serverID   string = "312292616089894924"
	xmark      string = "<:xmark:314349398824058880>"
)

var (
	dg           *discordgo.Session
	lastReboot   string
	log          = newLog()
	emojiRegex   = regexp.MustCompile("<(a)?:.*?:(.*?)>")
	userIDRegex  = regexp.MustCompile("<@!?([0-9]+)>")
	channelRegex = regexp.MustCompile("<#([0-9]+)>")
	status       = map[discordgo.Status]string{"dnd": "busy", "online": "online", "idle": "idle", "offline": "offline"}
	footer       = new(discordgo.MessageEmbedFooter)
)

func init() {
	footer.Text = "Created with ❤ by Strum355\nLast Bot reboot: " + time.Now().Format("Mon, 02-Jan-06 15:04:05 MST")
}

func dailyJobs() {
	for {
		postServerCount()
		time.Sleep(time.Hour * 24)
	}
}

func postServerCount() {
	url := "https://bots.discord.pw/api/bots/301819949683572738/stats"

	count := len(dg.State.Guilds)

	jsonStr := []byte(`{"server_count":` + strconv.Itoa(count) + `}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Error("error making bots.discord.pw request", err)
		return
	}

	req.Header.Set("Authorization", c.DiscordPWKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := new(http.Client).Do(req)
	if err != nil {
		log.Error("bots.discord.pw error", err)
		return
	}
	defer resp.Body.Close()

	log.Info("POSTed " + strconv.Itoa(count) + " to bots.discord.pw")

	if resp.StatusCode != http.StatusNoContent {
		log.Error("received " + strconv.Itoa(resp.StatusCode) + " from bots.discord.pw")
	}

}

func setBotGame(s *discordgo.Session) {
	if err := s.UpdateStatus(0, c.Game); err != nil {
		log.Error("Update status err:", err)
		return
	}
	log.Info("set initial game to", c.Game)
}

// Set all handlers for queued images, in case the bot crashes with images still in queue
func setQueuedImageHandlers() {
	for imgNum := range q.QueuedMsgs {
		imgNumInt, err := strconv.Atoi(imgNum)
		if err != nil {
			log.Error("Error converting string to num for queue:", err)
			continue
		}
		go fimageReview(dg, q, imgNumInt)
	}
}

func main() {
	runtime.GOMAXPROCS(c.MaxProc)

	log.Info("/*********BOT RESTARTING*********\\")

	names := []string{"config", "users", "servers", "queue"}
	for i, f := range []func() error{loadConfig, loadUsers, loadServers, loadQueue} {
		if err := f(); err != nil {
			switch i {
			case 0:
				os.Exit(404)
			default:
			}
			continue
		}
		log.Trace("loaded", names[i])
	}
	defer cleanup()

	log.Info("files loaded")

	var err error
	dg, err = discordgo.New("Bot " + c.Token)
	if err != nil {
		log.Error("Error creating Discord session,", err)
		return
	}

	log.Trace("session created")

	dg.AddHandler(messageCreateEvent)
	dg.AddHandler(presenceChangeEvent)
	dg.AddHandler(guildKickedEvent)
	dg.AddHandler(memberJoinEvent)
	dg.AddHandler(readyEvent)
	dg.AddHandler(guildJoinEvent)

	if err := dg.Open(); err != nil {
		log.Error("Error opening connection,", err)
		return
	}
	defer dg.Close()

	log.Trace("connection opened")

	sMap.Count = len(sMap.Server)

	go setQueuedImageHandlers()

	if !c.InDev {
		go dailyJobs()
	}

	// Setup http server for selfbots
	router := chi.NewRouter()
	router.Get("/image/{id:[0-9]{18}}/recall/{img:[0-9a-z]{64}}", httpImageRecall)
	router.Get("/inServer/{id:[0-9]{18}}", isInServer)

	go func() { log.Error("error starting http server", http.ListenAndServe("0.0.0.0:8080", router)) }()

	log.Info("Bot is now running. Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGSEGV, syscall.SIGHUP)
	<-sc
}
