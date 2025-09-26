package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	session    *discordgo.Session
	channelIds map[string]string
}

const channelName = "tournaments"

func InitBot(token string) (*Bot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Bot startup failed %s", err)
	}
	dg.ShouldReconnectOnError = true
	bot := &Bot{session: dg, channelIds: map[string]string{}}

	dg.AddHandler(func(_ *discordgo.Session, _ *discordgo.Ready) {
		log.Printf("Bot is ready")
	})

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.GuildCreate) {
		log.Printf("Guild \"%s\" added", r.Guild.Name)
		channels := r.Guild.Channels
		for _, channel := range channels {
			if channel.Name == channelName {
				bot.channelIds[channel.GuildID] = channel.ID
				log.Printf("Channel \"%s\" added for \"%s\"", channelName, r.Guild.Name)
			}
		}
	})

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.GuildDelete) {
		log.Printf("guild %s removed", r.Guild.Name)
		delete(bot.channelIds, r.Guild.ID)
	})

	err = dg.Open()
	if err != nil {
		return nil, err
	}
	return bot, nil
}

func (b *Bot) SendMessage(m string) {
	log.Print("SEND MESSAGE\n")
	for _, channelId := range b.channelIds {
		log.Printf("CHANNEL %s\n", channelId)
		_, err := b.session.ChannelMessageSend(channelId, m)
		if err != nil {
			log.Printf("Error while sending message to %s", channelId)
			return
		}
	}
}

func (b *Bot) Close() {
	b.session.Close()
}
