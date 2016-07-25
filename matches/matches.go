package matches

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tucnak/telebot"
)

var matches map[int64]*Match = map[int64]*Match{}

type Match struct {
	Day            string
	Month          string
	Year           string
	Hour           string
	Minutes        string
	Attendees      map[int]*Attendee
	AttendeesCount int
	Out            map[int]*Attendee
	Maybe          map[int]*Attendee
}

type Attendee struct {
	ID             int
	FirstName      string
	LastName       string
	Username       string
	Aditional      []string
	AditionalCount int
}

func (m *Match) UpdateAttendee(user telebot.User, cmd string, arg string) {
	switch cmd {
	case "/in":
		m.Attendees[user.ID] = &Attendee{ID: user.ID, FirstName: user.FirstName, LastName: user.LastName, Username: user.Username}
		delete(m.Out, user.ID)
		delete(m.Maybe, user.ID)
	case "/out":
		m.Out[user.ID] = &Attendee{ID: user.ID, FirstName: user.FirstName, LastName: user.LastName, Username: user.Username}
		delete(m.Attendees, user.ID)
		delete(m.Maybe, user.ID)
	case "/maybe":
		m.Maybe[user.ID] = &Attendee{ID: user.ID, FirstName: user.FirstName, LastName: user.LastName, Username: user.Username}
		delete(m.Attendees, user.ID)
		delete(m.Out, user.ID)
	}
}

func (m *Match) Status() string {
	msg := fmt.Sprintf("There's a Match scheduled for %s/%s/%s %s:%s:\nAttendees:", m.Day, m.Month, m.Year, m.Hour, m.Minutes)
	if len(m.Attendees) > 0 {
		for _, v := range m.Attendees {
			msg += "\n  - " + v.FirstName + " " + v.LastName
		}
	}
	msg += "\n"
	if len(m.Maybe) > 0 {
		msg += "Maybe:"
		for _, v := range m.Maybe {
			msg += "\n  - " + v.FirstName + " " + v.LastName
		}
		msg += "\n"
	}
	if len(m.Out) > 0 {
		msg += "Out:"
		for _, v := range m.Out {
			msg += "\n  - " + v.FirstName + " " + v.LastName
		}
		msg += "\n"
	}
	return msg
}

func NewMatch(groupId int64, date, t string) (*Match, error) {
	var day, month, year, hour, minutes string
	dateSplit := strings.Split(date, "/")
	if len(dateSplit) < 2 {
		return nil, fmt.Errorf("Date is invalid: %s", date)
	}
	if len(dateSplit) == 2 {
		year = strconv.Itoa(time.Now().Year())
		fmt.Printf("year: %s\n", year)
		day = dateSplit[0]
		month = dateSplit[1]
	}
	timeSplit := strings.Split(t, ":")
	if len(timeSplit) < 2 {
		return nil, fmt.Errorf("Time is invalid: %s", t)
	}
	hour = timeSplit[0]
	minutes = timeSplit[1]
	m := Match{
		Day:            day,
		Month:          month,
		Year:           year,
		Hour:           hour,
		Minutes:        minutes,
		Attendees:      make(map[int]*Attendee),
		AttendeesCount: 0,
		Out:            make(map[int]*Attendee),
		Maybe:          make(map[int]*Attendee),
	}
	matches[groupId] = &m
	return &m, nil
}

func MigrateGroup(from, to int64) {
	matches[to] = matches[from]
	delete(matches, from)
}

func GetMatch(groupId int64) *Match {
	return matches[groupId]
}
