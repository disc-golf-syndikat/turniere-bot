package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/resterle/turnire-bot/discord"
	"github.com/resterle/turnire-bot/turniere"
)

// Registration page
// https://discord.com/oauth2/authorize?client_id=1203403155933503569&scope=bot&permissions=2048

// NOTE: Players list
// https://turniere.discgolf.de/index.php?p=events&sp=list-players&id=2124

const baseUrl = "https://turniere.discgolf.de/index.php?p=events"
const discordTokenName = "DISCORD_TOKEN"
const schduleInterval = time.Hour
const notificationOffset = time.Hour * 10

var bot *discord.Bot
var turnaments = make(map[string]*turniere.Turnament)

func main() {
	token, tokenSet := os.LookupEnv(discordTokenName)
	if !tokenSet {
		log.Fatalf("%s needs to be set", discordTokenName)
	}

	var err error
	bot, err = discord.InitBot(token)
	if err != nil {
		log.Fatalf("Error while creating bot. %s", err)
	}
	// Wait here until CTRL-C or other term signal is received.
	log.Println("Bot is now running.  Press CTRL-C to exit.")

	//go schedule(bot)()
	ticker := time.NewTicker(schduleInterval)
	exit := make(chan int, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	go startScheduling(ticker.C, exit, &wg)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	log.Print("STOP")

	ticker.Stop()
	close(exit)

	// Cleanly close down the Discord session.
	bot.Close()
	wg.Wait()

	os.Exit(0)
}

func startScheduling(t <-chan time.Time, exit chan int, wg *sync.WaitGroup) {
	defer wg.Done()
	task()
	for {
		select {
		case <-t:
			log.Print("Scheduling task")
			task()
			continue
		case <-exit:
			log.Print("Stopping scheduler")
		}
		break
	}
}

func task() {
	reader, err := httpReader(baseUrl)
	//reader, err := fileReader("index.html")
	if err == nil {
		fetched := turniere.Parse(*reader)
		updated := mergeTurnaments(fetched)
		fetchDetails(updated)
		maybeSendMessages()
	}
}

func mergeTurnaments(fetched []turniere.Turnament) []string {
	updated := []string{}
	for i, ft := range fetched {
		t, present := turnaments[ft.Id]
		if !present || (present && ft.Changed.After(t.Changed)) {
			turnaments[ft.Id] = &fetched[i]
			updated = append(updated, ft.Id)
		}
	}
	return updated
}

func fetchDetails(ids []string) {
	for _, id := range ids {
		if turnaments[id].RegistrationStartDate != nil {
			time.Sleep(time.Millisecond * 500)
			log.Printf("Fetch details for %s # %v\n", turnaments[id].Title, turnaments[id].RegistrationStartDate)
			v := url.Values{}
			v.Set("sp", "view")
			v.Set("id", id)
			reader, err := httpReader(baseUrl + "&" + v.Encode())
			//reader, err := fileReader("details.html")
			if err == nil {
				phases := turniere.ParsePhases(*reader)
				p, _ := turnaments[id]
				p.Phases = append(p.Phases, phases...)
			}
		}
	}
}

func maybeSendMessages() {
	now := time.Now()
	for _, t := range turnaments {
		if t.RegistrationStartDate != nil {
			for i := range t.Phases {
				p := &t.Phases[i]
				d := p.RegistrationStartDate.Sub(now)
				if d < notificationOffset && d > 0 && !p.Notified {
					text := "⏰ Turnieranmeldung für \"**%s**\"\n%s\nöffnet **heute um %s Uhr**\n📍 %s\n🔗 %s"
					bot.SendMessage(fmt.Sprintf(text, t.Title, p.Title, p.RegistrationStartDate.Format("15:04"), t.Location, t.Link))
					p.Notified = true
				}
			}
		}
	}
}

// For development purposes
func fileReader(filename string) (io.Reader, error) {
	fi, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	return fi, nil
}

func httpReader(url string) (*io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error while fetching %s. %s", url, err)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error while fetching %s expected status code of %d got %d", url, http.StatusOK, resp.StatusCode)
		return nil, errors.New("status code error")
	}
	return &resp.Body, nil
}
