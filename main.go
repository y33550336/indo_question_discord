package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

type CVItem struct {
	AudioPath string
	Sentence  string
	Level     string
}

var cvItemsMap map[string][]CVItem
var currentCVItem *CVItem
var hintLevels map[string]int
var mistakeCounts map[string]int

func LoadCommonVoice(tsvPath string) ([]CVItem, error) {
	file, err := os.Open(tsvPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	log.Printf("Read %d records from %s", len(records), tsvPath)

	var items []CVItem
	for i, r := range records {
		if i == 0 {
			continue // header
		}
		if len(r) < 4 {
			continue
		}
		sentence := r[3]
		words := strings.Fields(sentence)

		if len(words) < 3 {
			continue
		}

		level := "normal"
		switch {
		case len(words) <= 5:
			level = "easy"
		case len(words) >= 10:
			level = "hard"
		}

		items = append(items, CVItem{
			AudioPath: "mcv-scripted-id-v24.0/cv-corpus-24.0-2025-12-05/id/clips/" + r[1],
			Sentence:  sentence,
			Level:     level,
		})
	}

	return items, nil
}

func loadCVItems() {
	items, err := LoadCommonVoice("mcv-scripted-id-v24.0/cv-corpus-24.0-2025-12-05/id/validated.tsv")
	if err != nil {
		log.Printf("Failed to load CV items: %v", err)
		return
	}
	cvItemsMap = make(map[string][]CVItem)
	for _, item := range items {
		cvItemsMap[item.Level] = append(cvItemsMap[item.Level], item)
	}
	log.Printf("Loaded CV items: easy=%d, normal=%d, hard=%d", len(cvItemsMap["easy"]), len(cvItemsMap["normal"]), len(cvItemsMap["hard"]))
}

func Normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "!", "")
	s = strings.ReplaceAll(s, "?", "")
	s = strings.ReplaceAll(s, ",", "")
	return strings.TrimSpace(s)
}

func Check(user, answer string) bool {
	return Normalize(user) == Normalize(answer)
}

func postCVQuestion(s *discordgo.Session, channelID string) {
	// 前の問題が未解決なら答えを表示
	if currentCVItem != nil {
		s.ChannelMessageSend(channelID, "前の問題が未解決でした。正解は: "+currentCVItem.Sentence)
	}

	var selectedItems []CVItem
	for _, items := range cvItemsMap {
		selectedItems = append(selectedItems, items...)
	}

	if len(selectedItems) == 0 {
		s.ChannelMessageSend(channelID, "No CV items loaded")
		return
	}

	item := selectedItems[rand.Intn(len(selectedItems))]
	currentCVItem = &item

	file, err := os.Open(item.AudioPath)
	if err != nil {
		s.ChannelMessageSend(channelID, "Error opening audio file")
		return
	}
	defer file.Close()

	s.ChannelFileSend(channelID, "listening.mp3", file)
	s.ChannelMessageSend(channelID, "🎯 本日の問題です！音声を聞いて文章を入力してください！")
}

func startDailyQuestion(s *discordgo.Session, channelID string) {
	for {
		now := time.Now()
		// 毎日9時に出題（JSTを想定）
		next := time.Date(now.Year(), now.Month(), now.Day(), 22, 41, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}

		duration := next.Sub(now)
		log.Printf("Next auto question in %v (at %v)", duration, next)

		time.Sleep(duration)
		postCVQuestion(s, channelID)
	}
}

func getMatchedWords(user, answer string) []string {
	u := Normalize(user)
	a := Normalize(answer)
	uwords := strings.Fields(u)
	awords := strings.Fields(a)
	seen := make(map[string]bool)
	set := make(map[string]bool)
	for _, w := range awords {
		seen[w] = true
	}
	var matched []string
	for _, w := range uwords {
		if seen[w] && !set[w] {
			matched = append(matched, w)
			set[w] = true
		}
	}
	return matched
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

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

	loadCVItems()
	hintLevels = make(map[string]int)
	mistakeCounts = make(map[string]int)

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot {
			return
		}

		if m.Content == "!ping" {
			s.ChannelMessageSend(m.ChannelID, "pong")
		}

		if strings.HasPrefix(m.Content, "!cv") {
			// 前の問題が未解決なら答えを表示
			if currentCVItem != nil {
				s.ChannelMessageSend(m.ChannelID, "前の問題が未解決でした。正解は: "+currentCVItem.Sentence)
			}
			hintLevels[m.Author.ID] = 0
			parts := strings.Fields(m.Content)
			level := "all"
			if len(parts) > 1 {
				level = parts[1]
			}
			var selectedItems []CVItem
			if level == "all" {
				for _, items := range cvItemsMap {
					selectedItems = append(selectedItems, items...)
				}
			} else {
				selectedItems = cvItemsMap[level]
			}
			if len(selectedItems) == 0 {
				s.ChannelMessageSend(m.ChannelID, "No CV items loaded for level: "+level)
				return
			}
			item := selectedItems[rand.Intn(len(selectedItems))]
			currentCVItem = &item
			file, err := os.Open(item.AudioPath)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "Error opening audio file")
				return
			}
			defer file.Close()
			s.ChannelFileSend(m.ChannelID, "listening.mp3", file)
			s.ChannelMessageSend(m.ChannelID, "Listen to the audio and type the sentence!")
		}

		if currentCVItem != nil && !strings.HasPrefix(m.Content, "!") {
			userInput := m.Content
			userID := m.Author.ID
			if Check(userInput, currentCVItem.Sentence) {
				response := "Correct! 🎉"
				s.ChannelMessageSend(m.ChannelID, response)
				mistakeCounts[userID] = 0
				currentCVItem = nil
				return
			}
			// 部分一致の単語を抽出
			matched := getMatchedWords(userInput, currentCVItem.Sentence)
			mistakeCounts[userID]++
			if len(matched) > 0 {
				msg := "部分一致した単語: " + strings.Join(matched, ", ") + "\n"
				if mistakeCounts[userID] >= 3 {
					msg += "不正解。正解は: " + currentCVItem.Sentence + "\n"
					s.ChannelMessageSend(m.ChannelID, msg)
					mistakeCounts[userID] = 0
					currentCVItem = nil
					return
				}
				remain := 3 - mistakeCounts[userID]
				msg += fmt.Sprintf("まだ不正解です。残り試行回数: %d", remain)
				s.ChannelMessageSend(m.ChannelID, msg)
			} else {
				if mistakeCounts[userID] >= 3 {
					s.ChannelMessageSend(m.ChannelID, "不正解。正解は: "+currentCVItem.Sentence+"\n")
					mistakeCounts[userID] = 0
					currentCVItem = nil
					return
				}
				remain := 3 - mistakeCounts[userID]
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("不正解です。残り試行回数: %d", remain))
			}
		}

		if m.Content == "!answer" {
			if currentCVItem == nil {
				s.ChannelMessageSend(m.ChannelID, "No current CV item. Use !cv first.")
				return
			}
			userID := m.Author.ID
			response := "回答: " + currentCVItem.Sentence
			s.ChannelMessageSend(m.ChannelID, response)
			mistakeCounts[userID] = 0
			hintLevels[userID] = 0
			currentCVItem = nil
		}

		if m.Content == "!hint" {
			if currentCVItem == nil {
				s.ChannelMessageSend(m.ChannelID, "No current CV item. Use !cv first.")
				return
			}
			userID := m.Author.ID
			level := hintLevels[userID]
			words := strings.Fields(currentCVItem.Sentence)
			var hint string
			switch level {
			case 0:
				hint = fmt.Sprintf("単語数: %d", len(words))
			case 1:
				charCounts := make([]string, len(words))
				charHints := make([]string, len(words))
				for i, w := range words {
					charCounts[i] = strconv.Itoa(len(w))
					charHints[i] = strings.Repeat("\\_", len(w))
				}
				hint = "単語の文字数: " + strings.Join(charCounts, ", ") + " " + strings.Join(charHints, " ")
			case 2:
				// 仮定の品詞: 全て名詞として
				pos := make([]string, len(words))
				for i := range pos {
					pos[i] = "名詞"
				}
				hint = "品詞: " + strings.Join(pos, ", ")
			case 3:
				initialHints := make([]string, len(words))
				for i, w := range words {
					if len(w) > 0 {
						initialHints[i] = string(w[0]) + strings.Repeat("\\_", len(w)-1)
					}
				}
				hint = "単語の冒頭: " + strings.Join(initialHints, " ")
			default:
				revealLevel := level - 3
				initialHints := make([]string, len(words))
				for i, w := range words {
					if len(w) > 0 {
						initialHints[i] = string(w[0]) + strings.Repeat("\\_", len(w)-1)
					}
				}
				if revealLevel < len(words) {
					hint = "最初の " + strconv.Itoa(revealLevel) + " 単語: " + strings.Join(words[:revealLevel], " ") + " " + strings.Join(initialHints[revealLevel:], " ")
				} else {
					hint = "全ての文が出ました 答え: " + currentCVItem.Sentence
					currentCVItem = nil
				}
			}
			s.ChannelMessageSend(m.ChannelID, hint)
			hintLevels[userID]++
		}
	})

	err = dg.Open()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Bot is running")

	// 自動出題を開始
	autoChannelID := os.Getenv("AUTO_QUESTION_CHANNEL_ID")
	if autoChannelID != "" {
		go startDailyQuestion(dg, autoChannelID)
		log.Println("Auto daily question enabled for channel:", autoChannelID)
	}

	// 終了待ち
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	dg.Close()
}
