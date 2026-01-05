package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Question struct {
	Type     string   `json:"type"`
	Question string   `json:"question"`
	Choices  []string `json:"choices,omitempty"`
	Answer   string   `json:"answer,omitempty"`
}

var questions []Question

func loadQuestions() {
	data, err := os.ReadFile("questions.json")
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(data, &questions)
}

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN が設定されていません")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}

	dg.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent

	rand.Seed(time.Now().UnixNano())
	loadQuestions()

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot {
			return
		}

		if m.Content == "!ping" {
			s.ChannelMessageSend(m.ChannelID, "pong")
		}

		if m.Content == "!today" {
			q := questions[rand.Intn(len(questions))]

			msg := "📘 今日の一問\n" + q.Question

			if q.Type == "vocab" && len(q.Choices) > 0 {
				for i, c := range q.Choices {
					msg += "\n" + string('A'+i) + ". " + c
				}
			}

			s.ChannelMessageSend(m.ChannelID, msg)
		}

	})

	err = dg.Open()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Bot is running")

	// 終了待ち
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	dg.Close()
}
