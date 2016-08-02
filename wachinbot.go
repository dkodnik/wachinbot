package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/sschepens/wachinbot/matches"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

var bot2 *tgbotapi.BotAPI

type InlineCallbackData struct {
	Command string `json:"c"`
	MatchID uint64 `json:"m"`
}

func stringCallbackData(cmd string, matchID uint64) string {
	data := InlineCallbackData{Command: cmd, MatchID: matchID}
	b, _ := json.Marshal(data)
	return string(b)
}

func main() {
	var err error
	bot2, err = tgbotapi.NewBotAPI(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates, err := bot2.GetUpdatesChan(updateConfig)

	for update := range updates {
		if update.Message != nil {
			processMessage(update.Message)
		} else if update.EditedMessage != nil {
			processEditedMessage(update.EditedMessage)
		} else if update.InlineQuery != nil {
			processInlineQuery(update.InlineQuery)
		} else if update.ChosenInlineResult != nil {
			processChosenInlineResult(update.ChosenInlineResult)
		} else if update.CallbackQuery != nil {
			processCallbackQuery(update.CallbackQuery)
		} else {
			log.Printf("received unknown update: %+v", update)
		}
	}
}

func generateMatchInlineKeyboard(match matches.Match) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("In", stringCallbackData("/in", match.ID)),
			tgbotapi.NewInlineKeyboardButtonData("Maybe", stringCallbackData("/maybe", match.ID)),
			tgbotapi.NewInlineKeyboardButtonData("Out", stringCallbackData("/out", match.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Refresh", stringCallbackData("/refresh", match.ID)),
			tgbotapi.NewInlineKeyboardButtonSwitch("Share", strconv.FormatUint(match.ID, 10)),
		),
	)
}

func processMessage(message *tgbotapi.Message) {
	if message.IsCommand() {
		command := message.Command()
		arguments := strings.Split(message.CommandArguments(), " ")
		switch command {
		case "start":
			help(message)
		case "help":
			help(message)
		case "match":
			if len(arguments) < 2 {
				msg := tgbotapi.NewMessage(message.Chat.ID, "Please specify a Date and a Time")
				msg.ReplyToMessageID = message.MessageID
				bot2.Send(msg)
			} else {
				match, err := matches.NewMatch(int64(message.From.ID), arguments[0], arguments[1])
				if err != nil {
					msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Error creating match: %s", err.Error()))
					msg.ReplyToMessageID = message.MessageID
					_, err = bot2.Send(msg)
					if err != nil {
						log.Println("Failed to reply to message: ", err)
					}
					return
				}
				status, err := match.Status()
				if err != nil {
					log.Println("Failed to get match status:", err)
				}
				msg := tgbotapi.NewMessage(message.Chat.ID, status)
				msg.ReplyMarkup = generateMatchInlineKeyboard(*match)
				_, err = bot2.Send(msg)
				if err != nil {
					log.Println("Failed to reply:", err)
				}
			}
		default:
			msg := tgbotapi.NewMessage(message.Chat.ID, "Invalid command")
			msg.ReplyToMessageID = message.MessageID
			_, err := bot2.Send(msg)
			if err != nil {
				log.Println("Failed to reply:", err)
			}
		}
	} else {
		fmt.Printf("Received unsupportted message: %+v\n", message)
	}
}

func processEditedMessage(message *tgbotapi.Message) {
}

func help(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID,
		`Hello! I'm Wachin your helper, my commands are:

/match Date Time - Creates a new Match
/status - Match status
/in - Join Match
/out - Leave Match
/maybe - Not sure

Be careful, I may steal you wife or wallet...`)
	msg.ReplyToMessageID = message.MessageID
	bot2.Send(msg)
}

func processCallbackQuery(callback *tgbotapi.CallbackQuery) {
	var data InlineCallbackData
	response := tgbotapi.CallbackConfig{CallbackQueryID: callback.ID}
	json.Unmarshal([]byte(callback.Data), &data)
	match, err := matches.GetMatch(data.MatchID)
	if err != nil {
		log.Println("Failed to get match:", err)
		bot2.AnswerCallbackQuery(tgbotapi.CallbackConfig{CallbackQueryID: callback.ID, Text: "Failed to find Match!"})
		return
	}

	switch data.Command {
	case "/in", "/maybe", "/out":
		switch data.Command {
		case "/in":
			response.Text = "Campeon!"
		case "/maybe":
			response.Text = "Pollera!"
		case "/out":
			response.Text = "Cagon!"
		}
		err = match.UpdateAttendee(callback.From, data.Command)
		if err != nil {
			log.Println("Failed to get update attendee status:", err)
			response.Text = "Failed to update attendance!"
		}
	case "/refresh":
		response.Text = "Refreshed!"
	}

	status, err := match.Status()
	if err != nil {
		log.Println("Failed to get match status:", err)
	}

	markup := generateMatchInlineKeyboard(*match)
	editMsg := tgbotapi.NewEditMessageText(0, 0, status)
	if callback.InlineMessageID != "" {
		editMsg.InlineMessageID = callback.InlineMessageID
	} else {
		editMsg.ChatID = callback.Message.Chat.ID
		editMsg.MessageID = callback.Message.MessageID
	}
	editMsg.ReplyMarkup = &markup
	_, err = bot2.Send(editMsg)
	if err != nil {
		log.Println("Failed to edit message:", err)
	}

	matchMsgs, err := matches.GetMatchMessages(match.ID)
	if err != nil {
		log.Println("Failed to get match messages: ", err)
	}

	for _, mm := range matchMsgs {
		fmt.Printf("Updating inline message: %s\n", mm.InlineMessageID)
		if mm.InlineMessageID != editMsg.InlineMessageID {
			mkp := generateMatchInlineKeyboard(*match)
			eMsg := tgbotapi.NewEditMessageText(0, 0, status)
			eMsg.ReplyMarkup = &mkp
			eMsg.InlineMessageID = mm.InlineMessageID
			_, err = bot2.Send(eMsg)
			if err != nil {
				log.Println("Failed to edit message:", err)
			}
		}
	}

	_, err = bot2.AnswerCallbackQuery(response)
	if err != nil {
		log.Println("Failed to respond to query:", err)
	}
}

func processInlineQuery(query *tgbotapi.InlineQuery) {
	response := tgbotapi.InlineConfig{
		InlineQueryID: query.ID,
		IsPersonal:    true,
		CacheTime:     0,
	}
	defer func() {
		_, err := bot2.AnswerInlineQuery(response)
		if err != nil {
			log.Println("Failed to respond to query:", err)
		}
	}()

	if len(strings.TrimSpace(query.Query)) == 0 {
		matchesResult, err := matches.GetMatches(int64(query.From.ID))
		if err != nil {
			fmt.Println("Error getting matches: ", err)
			return
		}

		for _, m := range matchesResult {
			status, _ := m.Status()
			article := tgbotapi.NewInlineQueryResultArticle(strconv.FormatUint(m.ID, 10), "Match "+m.Day+"/"+m.Month+" "+m.Hour+":"+m.Minutes, status)
			markup := generateMatchInlineKeyboard(m)
			article.ReplyMarkup = &markup
			response.Results = append(response.Results, article)
		}
	} else {
		id, err := strconv.ParseUint(strings.TrimSpace(query.Query), 10, 0)
		if err != nil {
			return
		}
		m, err := matches.GetMatch(id)
		if err != nil {
			fmt.Println("Error getting matches: ", err)
			return
		}

		status, _ := m.Status()
		article := tgbotapi.NewInlineQueryResultArticle(strconv.FormatUint(m.ID, 10), "Match "+m.Day+"/"+m.Month+" "+m.Hour+":"+m.Minutes, status)
		markup := generateMatchInlineKeyboard(*m)
		article.ReplyMarkup = &markup
		response.Results = append(response.Results, article)
	}
}

func processChosenInlineResult(result *tgbotapi.ChosenInlineResult) {
	matchID, err := strconv.ParseUint(result.ResultID, 10, 0)
	if err != nil {
		fmt.Println("Error parsing match id: ", err)
		return
	}

	_, err = matches.CreateMatchMessage(matchID, result.InlineMessageID)
	if err != nil {
		fmt.Println("Error creating MatchMessage: ", err)
		return
	}
}
