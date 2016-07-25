package main

import (
	"fmt"
	"github.com/sschepens/wachinbot/matches"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tucnak/telebot"
)

var bot *telebot.Bot

func main() {
	var err error
	bot, err = telebot.NewBot(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}

	bot.Messages = make(chan telebot.Message, 1000)
	bot.Queries = make(chan telebot.Query, 1000)

	go messages()
	go queries()

	bot.Start(1 * time.Second)
}

func messages() {
	for message := range bot.Messages {
		log.Printf("Received a message from %s with the text: %s\n", message.Sender.Username, message.Text)
		if strings.HasPrefix(message.Text, "/") {
			arguments := strings.Split(message.Text, " ")
			command := arguments[0]
			if strings.Contains(message.Text, "@") {
				command = strings.Split(command, "@")[0]
			}
			switch command {
			case "/start":
				help(bot, message)
			case "/help":
				help(bot, message)
			case "/match":
				if len(arguments) < 3 {
					bot.SendMessage(message.Chat, "Please specify a Date and a Time", &telebot.SendOptions{ReplyTo: message})
				} else {
					_, err := matches.NewMatch(message.Chat.ID, arguments[1], arguments[2])
					if err != nil {
						bot.SendMessage(message.Chat, fmt.Sprintf("Error creating match: %s", err.Error()), &telebot.SendOptions{ReplyTo: message})
						continue
					}
					bot.SendMessage(message.Chat, fmt.Sprintf("Match created on Date %s and Time %s", arguments[1], arguments[2]), &telebot.SendOptions{ReplyTo: message})
				}
			case "/status":
				var match = matches.GetMatch(message.Chat.ID)
				if match == nil {
					bot.SendMessage(message.Chat, "You have no match scheduled", &telebot.SendOptions{ReplyTo: message})
					continue
				}
				bot.SendMessage(message.Chat, match.Status(), &telebot.SendOptions{ReplyTo: message})
			case "/in", "/out", "/maybe":
				var match = matches.GetMatch(message.Chat.ID)
				if match == nil {
					bot.SendMessage(message.Chat, "You have no match scheduled", &telebot.SendOptions{ReplyTo: message})
					continue
				}
				match.UpdateAttendee(message.Sender, command, "")
				if command == "/out" {
					bot.SendMessage(message.Chat, "Gay", &telebot.SendOptions{ReplyTo: message})
					continue
				}
			default:
				bot.SendMessage(message.Chat, "Invalid command", &telebot.SendOptions{ReplyTo: message})
			}
		} else {
			bot.SendMessage(message.Chat, "Gay", &telebot.SendOptions{ReplyTo: message})
		}
	}
}

func help(bot *telebot.Bot, message telebot.Message) {
	bot.SendMessage(message.Chat,
		`Hello! I'm Wachin your helper, my commands are:

/match Date Time - Creates a new Match
/status - Match status
/in - Join Match
/out - Leave Match
/maybe - Not sure

Be careful, I may steal you wife or wallet...`,
		&telebot.SendOptions{ReplyTo: message})
}

func queries() {
	for query := range bot.Queries {
		log.Println("--- new query ---")
		log.Println("from:", query.From.Username)
		log.Println("text:", query.Text)

		// Create an article (a link) object to show in our results.
		article := &telebot.InlineQueryResultArticle{
			Title: "Telegram bot framework written in Go",
			URL:   "https://github.com/tucnak/telebot",
			InputMessageContent: &telebot.InputTextMessageContent{
				Text:           "Telebot is a convenient wrapper to Telegram Bots API, written in Golang.",
				DisablePreview: false,
			},
		}

		// Build the list of results. In this instance, just our 1 article from above.
		results := []telebot.InlineQueryResult{article}

		// Build a response object to answer the query.
		response := telebot.QueryResponse{
			Results:    results,
			IsPersonal: true,
		}

		// And finally send the response.
		if err := bot.AnswerInlineQuery(&query, &response); err != nil {
			log.Println("Failed to respond to query:", err)
		}
	}
}
