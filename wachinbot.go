package main

import (
	"fmt"
	"github.com/sschepens/wachinbot/matches"
	"log"
	"os"
	"strings"
	"time"
	"encoding/json"

	"github.com/sschepens/telebot"
)

var bot *telebot.Bot

type InlineCallbackData struct {
	Command   string `json:"c"`
	MatchID uint64 `json:"m"`
}

func stringCallbackData(cmd string, matchID uint64) string {
	data := InlineCallbackData{Command: cmd, MatchID: matchID}
	b, _ := json.Marshal(data)
	return string(b)
}

func main() {
	var err error
	bot, err = telebot.NewBot(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}

	bot.Messages = make(chan telebot.Message, 1000)
	bot.Queries = make(chan telebot.Query, 1000)
	bot.Callbacks = make(chan telebot.Callback, 1000)

	go messages()
	go queries()
	go callbacks()

	bot.Start(1 * time.Second)
}

func messages() {
	for message := range bot.Messages {
		log.Printf("Received a message from %s with the text: %s\n", message.Sender.Username, message.Text)
		if strings.HasPrefix(message.Text, "/") {
			arguments := strings.Split(message.Text, " ")
			command := arguments[0]
			if strings.Contains(message.Text, "@") {
				commandSplit := strings.Split(command, "@")
				if commandSplit[1] != "wachinbot" {
					continue
				}
				command = commandSplit[0]
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
					match, err := matches.NewMatch(message.Origin().ID, arguments[1], arguments[2])
					if err != nil {
						bot.SendMessage(message.Chat, fmt.Sprintf("Error creating match: %s", err.Error()), &telebot.SendOptions{ReplyTo: message})
						continue
					}
					status, err := match.Status()
					if err != nil {
						log.Println("Failed to get match status:", err)
					}
					_, err = bot.SendMessage(message.Chat, status, &telebot.SendOptions{ReplyMarkup: telebot.ReplyMarkup{
						InlineKeyboard: [][]telebot.KeyboardButton{
							[]telebot.KeyboardButton{
								telebot.KeyboardButton{Text: "In", Data: stringCallbackData("/in", match.ID)},
								telebot.KeyboardButton{Text: "Maybe", Data: stringCallbackData("/maybe", match.ID)},
								telebot.KeyboardButton{Text: "Out", Data: stringCallbackData("/out", match.ID)},
							},
							[]telebot.KeyboardButton{
								telebot.KeyboardButton{Text: "Refresh Status", Data: stringCallbackData("/refresh", match.ID)},
							},
						},
					}})
					if err != nil {
						log.Println("Failed to reply:", err)
					}
				}
			default:
				bot.SendMessage(message.Chat, "Invalid command", &telebot.SendOptions{ReplyTo: message})
			}
		} else {
			fmt.Printf("Received unsupportted message: %+v\n", message)
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
		matchesResult, err := matches.GetMatches(query.From.ID)
		if err != nil {
			fmt.Println("Error getting matches: ", err)
			continue
		}

		var results []telebot.InlineQueryResult

		for _, m := range matchesResult {
			status, _ := m.Status()
			article := &telebot.InlineQueryResultArticle{
				Title: "Match " + m.Day + "/" + m.Month + " " +m.Hour + ":"+m.Minutes,
				InputMessageContent: &telebot.InputTextMessageContent{
					Text:  status,
					DisablePreview: true,
				},
				ReplyMarkup: telebot.InlineKeyboardMarkup{
					InlineKeyboard: [][]telebot.KeyboardButton{
						[]telebot.KeyboardButton{
							telebot.KeyboardButton{Text: "In", Data: stringCallbackData("/in", m.ID)},
							telebot.KeyboardButton{Text: "Maybe", Data: stringCallbackData("/maybe", m.ID)},
							telebot.KeyboardButton{Text: "Out", Data: stringCallbackData("/out", m.ID)},
						},
						[]telebot.KeyboardButton{
							telebot.KeyboardButton{Text: "Refresh Status", Data: stringCallbackData("/refresh", m.ID)},
						},
					},
				},
			}
			results = append(results, article)
		}

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

func callbacks() {
	for callback := range bot.Callbacks {
		var data InlineCallbackData
		var response *telebot.CallbackResponse
		json.Unmarshal([]byte(callback.Data), &data)
		match, err := matches.GetMatch(data.MatchID)
		if err != nil {
			log.Println("Failed to get match:", err)
			bot.AnswerCallbackQuery(&callback, &telebot.CallbackResponse{Text: "Failed to find Match!"})
			continue
		}

		switch data.Command {
		case "/in", "/maybe", "/out":
			switch data.Command {
			case "/in":
				response = &telebot.CallbackResponse{Text: "Campeon!"}
			case "/maybe":
				response = &telebot.CallbackResponse{Text: "Pollera!"}
			case "/out":
				response = &telebot.CallbackResponse{Text: "Cagon!"}
			}
			err = match.UpdateAttendee(callback.Sender, data.Command)
			if err != nil {
				log.Println("Failed to get update attendee status:", err)
				response = &telebot.CallbackResponse{Text: "Failed to update attendance!"}
			}
		case "/refresh":
			response = &telebot.CallbackResponse{Text: "Refreshed!"}
		default:
			response = &telebot.CallbackResponse{}
		}

		status, err := match.Status()
		if err != nil {
			log.Println("Failed to get match status:", err)
		}
		sendOptions := &telebot.SendOptions{ReplyMarkup: telebot.ReplyMarkup{
			InlineKeyboard: [][]telebot.KeyboardButton{
				[]telebot.KeyboardButton{
					telebot.KeyboardButton{Text: "In", Data: stringCallbackData("/in", match.ID)},
					telebot.KeyboardButton{Text: "Maybe", Data: stringCallbackData("/maybe", match.ID)},
					telebot.KeyboardButton{Text: "Out", Data: stringCallbackData("/out", match.ID)},
				},
				[]telebot.KeyboardButton{
					telebot.KeyboardButton{Text: "Refresh Status", Data: stringCallbackData("/refresh", match.ID)},
				},
			},
		}}
		if callback.MessageID != "" {
			err = bot.EditInlineMessageText(callback.MessageID, status, sendOptions)
		} else {
			_, err = bot.EditMessageText(callback.Message, status, sendOptions)
		}
		if err != nil {
			log.Println("Failed to reply callback:", err)
		}

		err = bot.AnswerCallbackQuery(&callback, response)
		if err != nil {
			log.Println("Failed to respond to query:", err)
		}
	}
}
