package matches

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

var db *gorm.DB

func init() {
	var err error
	db, err = gorm.Open("sqlite3", "./wachin.db")
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&Match{})
	db.AutoMigrate(&Attendee{})
	db.AutoMigrate(&MatchMessage{})
	db.LogMode(true)
}

type Match struct {
	ID        uint64 `gorm:"primary_key"`
	UserID    int64  `gorm:"index"`
	Attendees []Attendee
	Time      time.Time
}

type Attendee struct {
	ID        uint64 `gorm:"primary_key"`
	UserID    int64  `gorm:"index"`
	MatchID   uint64 `gorm:"index:idx_match_id_status"`
	FirstName string
	LastName  string
	Username  string
	Status    string `gorm:"index:idx_match_id_status"`
}

type MatchMessage struct {
	ID              uint64 `gorm:"primary_key"`
	MatchID         uint64 `gorm:"index"`
	InlineMessageID string
}

type MatchStatus struct {
	In    []string
	Out   []string
	Maybe []string
}

func (m *Match) In() ([]Attendee, error) {
	var attendees []Attendee
	for _, a := range m.Attendees {
		if a.Status == "maybe" {
			attendees = append(attendees, a)
		}
	}
	return attendees, nil
}

func (m *Match) Out() ([]Attendee, error) {
	var attendees []Attendee
	for _, a := range m.Attendees {
		if a.Status == "out" {
			attendees = append(attendees, a)
		}
	}
	return attendees, nil
}

func (m *Match) Maybe() ([]Attendee, error) {
	var attendees []Attendee
	for _, a := range m.Attendees {
		if a.Status == "maybe" {
			attendees = append(attendees, a)
		}
	}
	return attendees, nil
}

func (m *Match) UpdateAttendee(user *tgbotapi.User, cmd string) error {
	var attendee Attendee
	status := cmd[1:]
	err := db.First(&attendee, "user_id = ? AND match_id = ?", user.ID, m.ID)
	notFound := err.RecordNotFound()
	if err.Error != nil && !notFound {
		return err.Error
	}

	attendee.Status = status
	if notFound {
		attendee.FirstName = user.FirstName
		attendee.LastName = user.LastName
		attendee.Username = user.UserName
		attendee.UserID = int64(user.ID)
		attendee.MatchID = m.ID
		err = db.Create(&attendee)
	} else {
		err = db.Save(&attendee)
	}
	if err.Error != nil {
		return err.Error
	}

	return nil
}

func (m *Match) AddExternalAttendee(p string) error {
	if p == "" {
		return errors.New("emmpty player")
	}
	player := strings.Title(strings.ToLower(p))
	var attendee Attendee
	err := db.First(&attendee, "first_name = ? AND last_name = ? AND match_id = ?", player, "", m.ID)
	notFound := err.RecordNotFound()
	if err.Error != nil && !notFound {
		return err.Error
	}

	attendee.Status = "in"
	if notFound {
		attendee.FirstName = player
		attendee.MatchID = m.ID
		err = db.Create(&attendee)
	} else {
		err = db.Save(&attendee)
	}
	if err.Error != nil {
		return err.Error
	}

	return nil
}

func (m *Match) RemoveExternalAttendee(p string) error {
	if p == "" {
		return errors.New("emmpty player")
	}
	player := strings.Title(strings.ToLower(p))
	var attendee Attendee
	err := db.First(&attendee, "first_name = ? AND last_name = ? AND match_id = ?", player, "", m.ID)
	if err.Error != nil {
		if err.RecordNotFound() {
			return errors.New("player not found")
		}
		return err.Error
	}

	err = db.Delete(&attendee)

	if err.Error != nil {
		return err.Error
	}

	return nil
}

func (m *Match) FormatTime() string {
	return m.Time.Format("Mon, 02 Jan 15:04")
}

func (m *Match) Status() (msg string, err error) {
	var attendees []Attendee
	var matchStatus MatchStatus
	err = db.Find(&attendees, "match_id = ?", m.ID).Error
	if err != nil {
		return
	}
	for _, a := range attendees {
		name := a.FirstName + " " + a.LastName
		switch a.Status {
		case "in":
			matchStatus.In = append(matchStatus.In, name)
		case "out":
			matchStatus.Out = append(matchStatus.Out, name)
		case "maybe":
			matchStatus.Maybe = append(matchStatus.Maybe, name)
		}
	}
	msg = fmt.Sprintf("Match #%d on %s:\n", m.ID, m.FormatTime())
	if len(matchStatus.In) > 0 {
		msg += "Attendees: "
		msg += strconv.Itoa(len(matchStatus.In))
		for _, v := range matchStatus.In {
			msg += "\n  - " + v
		}
		msg += "\n"
	}
	if len(matchStatus.Maybe) > 0 {
		msg += "Maybe: "
		msg += strconv.Itoa(len(matchStatus.Maybe))
		for _, v := range matchStatus.Maybe {
			msg += "\n  - " + v
		}
		msg += "\n"
	}
	if len(matchStatus.Out) > 0 {
		msg += "Out: "
		msg += strconv.Itoa(len(matchStatus.Out))
		for _, v := range matchStatus.Out {
			msg += "\n  - " + v
		}
		msg += "\n"
	}
	return
}

func NewMatch(userID int64, date, t string) (*Match, error) {
	var day, month, year, hour, minutes int
	dateSplit := strings.Split(date, "/")
	if len(dateSplit) < 2 {
		return nil, fmt.Errorf("Date is invalid: %s", date)
	}
	if len(dateSplit) == 2 {
		year = time.Now().Year()
		day, _ = strconv.Atoi(dateSplit[0])
		month, _ = strconv.Atoi(dateSplit[1])
	}
	timeSplit := strings.Split(t, ":")
	if len(timeSplit) < 2 {
		return nil, fmt.Errorf("Time is invalid: %s", t)
	}
	hour, _ = strconv.Atoi(timeSplit[0])
	minutes, _ = strconv.Atoi(timeSplit[1])
	dateTime := time.Date(year, time.Month(month), day, hour, minutes, 0, 0, time.Local)
	if dateTime.Before(time.Now()) {
		return nil, errors.New("Time is before than now")
	}
	m := Match{
		UserID: userID,
		Time:   dateTime,
	}
	err := db.Create(&m)
	if err.Error != nil {
		return nil, err.Error
	}
	return &m, nil
}

func GetMatch(userID int64, id uint64) (*Match, error) {
	var match Match
	err := db.Preload("Attendees").Joins("LEFT OUTER JOIN attendees on attendees.match_id = matches.id").Select("DISTINCT matches.*").Where("(matches.user_id = ? OR attendees.user_id = ?) AND matches.time > ?", userID, userID, time.Now()).First(&match, id)
	return &match, err.Error
}

func GetMatches(userID int64) ([]Match, error) {
	var matches []Match
	err := db.Preload("Attendees").Joins("LEFT OUTER JOIN attendees on attendees.match_id = matches.id").Select("DISTINCT matches.*").Find(&matches, "(matches.user_id = ? OR attendees.user_id = ?) AND matches.time > ?", userID, userID, time.Now())
	return matches, err.Error
}

func CreateMatchMessage(matchID uint64, msgID string) (*MatchMessage, error) {
	matchMsg := MatchMessage{
		MatchID:         matchID,
		InlineMessageID: msgID,
	}
	err := db.Create(&matchMsg)
	if err.Error != nil {
		return nil, err.Error
	}
	return &matchMsg, nil
}

func GetMatchMessages(matchID uint64) ([]MatchMessage, error) {
	var matchMsgs []MatchMessage
	err := db.Find(&matchMsgs, "match_id = ?", matchID).Error
	if err != nil {
		return nil, err
	}
	return matchMsgs, nil
}
