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

func generateMatchPublicInlineKeyboard(match matches.Match) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("In", stringCallbackData("/in", match.ID)),
			tgbotapi.NewInlineKeyboardButtonData("Maybe", stringCallbackData("/maybe", match.ID)),
			tgbotapi.NewInlineKeyboardButtonData("Out", stringCallbackData("/out", match.ID)),
		),
	)
}

func generateMatchPrivateInlineKeyboard(match matches.Match) tgbotapi.InlineKeyboardMarkup {
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
		arguments := strings.Split(strings.TrimSpace(message.CommandArguments()), " ")
		switch command {
		case "start":
			help(message)
		case "help":
			help(message)
		case "add", "remove":
			argLen := len(arguments)
			if argLen < 1 {
				msg := tgbotapi.NewMessage(message.Chat.ID, "Please specify a Match and a Player")
				msg.ReplyToMessageID = message.MessageID
				bot2.Send(msg)
			} else if argLen < 2 {
				msg := tgbotapi.NewMessage(message.Chat.ID, "Please specify a Player")
				msg.ReplyToMessageID = message.MessageID
				bot2.Send(msg)
			} else {
				id, err := strconv.ParseUint(arguments[0], 10, 0)
				if err != nil {
					msg := tgbotapi.NewMessage(message.Chat.ID, "Invalid Match number")
					msg.ReplyToMessageID = message.MessageID
					bot2.Send(msg)
				} else {
					m, err := matches.GetMatch(int64(message.From.ID), id)
					if err != nil {
						msg := tgbotapi.NewMessage(message.Chat.ID, "Match not found")
						msg.ReplyToMessageID = message.MessageID
						bot2.Send(msg)
					} else {
						if command == "add" {
							err = m.AddExternalAttendee(strings.Join(arguments[1:], " "))
						} else {
							err = m.RemoveExternalAttendee(strings.Join(arguments[1:], " "))
						}
						if err != nil {
							msg := tgbotapi.NewMessage(message.Chat.ID, "Error: "+err.Error())
							msg.ReplyToMessageID = message.MessageID
							bot2.Send(msg)
						} else {
							updateInlineMatchMessages(m, "")
						}
					}
				}
			}
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
				msg.ReplyMarkup = generateMatchPrivateInlineKeyboard(*match)
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

Be careful, I may steal you wife or wallet...`)
	msg.ReplyToMessageID = message.MessageID
	bot2.Send(msg)
}

func processCallbackQuery(callback *tgbotapi.CallbackQuery) {
	var data InlineCallbackData
	response := tgbotapi.CallbackConfig{CallbackQueryID: callback.ID}
	json.Unmarshal([]byte(callback.Data), &data)
	match, err := matches.GetMatch(int64(callback.From.ID), data.MatchID)
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

	var markup tgbotapi.InlineKeyboardMarkup
	editMsg := tgbotapi.NewEditMessageText(0, 0, status)
	if callback.InlineMessageID != "" {
		editMsg.InlineMessageID = callback.InlineMessageID
		markup = generateMatchPublicInlineKeyboard(*match)
	} else {
		editMsg.ChatID = callback.Message.Chat.ID
		editMsg.MessageID = callback.Message.MessageID
		markup = generateMatchPrivateInlineKeyboard(*match)
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
			mkp := generateMatchPublicInlineKeyboard(*match)
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

func updateChatMatchMessage(match *matches.Match, chatID int64, messageID int) {
	var markup tgbotapi.InlineKeyboardMarkup
	status, err := match.Status()
	if err != nil {
		log.Println("Failed to get match status:", err)
	}
	editMsg := tgbotapi.NewEditMessageText(0, 0, status)
	editMsg.ChatID = chatID
	editMsg.MessageID = messageID
	markup = generateMatchPrivateInlineKeyboard(*match)
	editMsg.ReplyMarkup = &markup
	_, err = bot2.Send(editMsg)
	if err != nil {
		log.Println("Failed to edit message:", err)
	}
}

func updateInlineMatchMessages(match *matches.Match, inlineMessageID string) {
	status, err := match.Status()
	if err != nil {
		log.Println("Failed to get match status:", err)
	}

	var markup tgbotapi.InlineKeyboardMarkup
	if inlineMessageID != "" {
		editMsg := tgbotapi.NewEditMessageText(0, 0, status)
		editMsg.InlineMessageID = inlineMessageID
		markup = generateMatchPublicInlineKeyboard(*match)
		editMsg.ReplyMarkup = &markup
		_, err = bot2.Send(editMsg)
		if err != nil {
			log.Println("Failed to edit message:", err)
		}
	}

	matchMsgs, err := matches.GetMatchMessages(match.ID)
	if err != nil {
		log.Println("Failed to get match messages: ", err)
	}

	for _, mm := range matchMsgs {
		fmt.Printf("Updating inline message: %s\n", mm.InlineMessageID)
		if mm.InlineMessageID != inlineMessageID {
			mkp := generateMatchPublicInlineKeyboard(*match)
			eMsg := tgbotapi.NewEditMessageText(0, 0, status)
			eMsg.ReplyMarkup = &mkp
			eMsg.InlineMessageID = mm.InlineMessageID
			_, err = bot2.Send(eMsg)
			if err != nil {
				log.Println("Failed to edit message:", err)
			}
		}
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
		fmt.Println("list all matches")
		matchesResult, err := matches.GetMatches(int64(query.From.ID))
		if err != nil {
			fmt.Println("Error getting matches: ", err)
			return
		}

		for _, m := range matchesResult {
			fmt.Printf("match: %d", m.ID)
			status, _ := m.Status()
			article := tgbotapi.NewInlineQueryResultArticle(strconv.FormatUint(m.ID, 10), fmt.Sprintf("Match #%d on %s", m.ID, m.FormatTime()), status)
			markup := generateMatchPublicInlineKeyboard(m)
			article.ReplyMarkup = &markup
			response.Results = append(response.Results, article)
		}
	} else {
		fmt.Printf("find matches '%s'", query.Query)
		querySplit := strings.Split(strings.TrimSpace(query.Query), " ")
		id, err := strconv.ParseUint(querySplit[0], 10, 0)
		if err != nil {
			return
		}
		m, err := matches.GetMatch(int64(query.From.ID), id)
		if err != nil {
			fmt.Println("Error getting matches: ", err)
			return
		}

		player := strings.TrimSpace(strings.Join(querySplit[1:], " "))

		var article tgbotapi.InlineQueryResultArticle
		if len(player) == 0 {
			status, _ := m.Status()
			article = tgbotapi.NewInlineQueryResultArticle(strconv.FormatUint(m.ID, 10), fmt.Sprintf("Match #%d on %s", m.ID, m.FormatTime()), status)
			markup := generateMatchPublicInlineKeyboard(*m)
			article.ReplyMarkup = &markup
			response.Results = append(response.Results, article)
		}

		add := fmt.Sprintf("Add player '%s' to Match #%d", player, id)
		article = tgbotapi.NewInlineQueryResultArticle(add, add, fmt.Sprintf("/add@wachinbot %d %s", id, player))
		response.Results = append(response.Results, article)

		remove := fmt.Sprintf("Remove player '%s' from Match #%d", player, id)
		article = tgbotapi.NewInlineQueryResultArticle(remove, remove, fmt.Sprintf("/remove@wachinbot %d %s", id, player))
		response.Results = append(response.Results, article)
	}
}

func processChosenInlineResult(result *tgbotapi.ChosenInlineResult) {
	if result.ResultID == "add" || result.ResultID == "remove" {
		return
	}
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
