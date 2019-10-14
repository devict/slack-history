package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var users map[string]User

type Channel struct {
	Name       string   `json:"name"`
	IsArchived bool     `json:"is_archived"`
	Members    []string `json:"members"`

	Count       int
	Percent     float64
	Characters  int
	CharPercent float64
	LastMessage time.Time
}

var channels map[string]Channel

var sorted Users

var (
	total      int
	totalChars int
)

var (
	sortM    = flag.String("sort", "messages", "Sort method. Options are messages, characters, or verbosity")
	fromS    = flag.String("from", "", "Start date for analysis in YYYY-MM-DD form. Blank for all time.")
	from     time.Time
	toS      = flag.String("to", "", "End date for analysis in YYYY-MM-DD form. Blank for all time.")
	to       time.Time
	excludeS = flag.String("exclude", "", "Comma separated list of users to exclude")
	exclude  map[string]struct{}
	limit    = flag.Int("limit", 20, "Limit the number of users to show")
	channel  = flag.String("channel", "", "Only count a certain channel. Blank for all.")
	user     = flag.String("user", "", "Only count a certain user. Blank for all.")
	dir      = flag.String("dir", ".", "Directory with export data.")
)

func main() {
	parseFlags()
	parseUsers()
	parseMessages()
	sortMessages()
	display()
}

func parseFlags() {
	flag.Parse()

	if *limit <= 0 {
		log.Fatal("-limit must be positive")
	}

	os.Chdir(*dir)

	if *fromS != "" {
		var err error
		if from, err = time.Parse("2006-01-02", *fromS); err != nil {
			log.Fatal("Could not parse -from time.", err)
		}
	}

	if *toS != "" {
		var err error
		if to, err = time.Parse("2006-01-02", *toS); err != nil {
			log.Fatal("Could not parse -to time.", err)
		}
	}

	if *excludeS != "" {
		exclude = make(map[string]struct{})
		for _, u := range strings.Split(*excludeS, ",") {
			exclude[u] = struct{}{}
		}
	}
}

func parseUsers() {
	u, err := os.Open("users.json")
	if err != nil {
		log.Fatal(err)
	}

	var raw []User
	if err := json.NewDecoder(u).Decode(&raw); err != nil {
		log.Fatal(err)
	}

	users = make(map[string]User)
	for _, u := range raw {
		if u.ID == "" {
			continue
		}
		if _, ok := exclude[u.Name]; ok {
			continue
		}
		users[u.ID] = u
	}
}

func parseMessages() {
	channels = make(map[string]Channel)

	var chs []Channel
	f, err := os.Open("channels.json")
	if err != nil {
		log.Fatal(err)
	}

	if err := json.NewDecoder(f).Decode(&chs); err != nil {
		log.Fatal(err)
	}

	for _, c := range chs {
		if c.IsArchived {
			continue
		}
		processChannel(c)
	}

	// Do post processing for user percentages
	for id, u := range users {
		u.Percent = float64(u.Count) / float64(total)
		u.CharPercent = float64(u.Characters) / float64(totalChars)
		users[id] = u
	}

	// Do some post processing for channel percentages
	for name, c := range channels {
		c.Percent = float64(c.Count) / float64(total)
		c.CharPercent = float64(c.Characters) / float64(totalChars)
		channels[name] = c
	}
}

func processChannel(c Channel) {
	if *channel != "" && c.Name != *channel {
		return
	}
	days, err := filepath.Glob(filepath.Join(c.Name, "*"))
	if err != nil {
		log.Fatal(err)
	}

	channels[c.Name] = c

	for _, day := range days {
		processDay(c.Name, day)
	}
}

func processDay(channel, path string) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var msgs []struct {
		Type    string `json:"type"`
		SubType string `json:"subtype"`
		User    string `json:"user"`
		TS      string `json:"ts"`
		Text    string `json:"text"`
	}

	if err := json.NewDecoder(f).Decode(&msgs); err != nil {
		log.Fatal(err)
	}

	c := channels[channel]

	for _, m := range msgs {

		// TODO sometimes the bots talk but don't have the bot_message subtype
		if m.Type != "message" || contains([]string{"channel_join", "channel_leave", "bot_message", "channel_purpose", "channel_topic"}, m.SubType) {
			continue
		}
		if m.User == "" {
			continue
		}

		ts, err := strconv.ParseFloat(m.TS, 64)
		if err != nil {
			log.Fatal(err)
		}

		if !from.IsZero() && from.Unix() > int64(ts) {
			continue
		}

		if !to.IsZero() && to.Unix() < int64(ts) {
			continue
		}

		u, present := users[m.User]
		if !present {
			continue
		}

		// TODO move this up?
		when := time.Unix(int64(ts), 0)
		if when.After(c.LastMessage) {
			c.LastMessage = when
		}

		c.Count++
		u.Count++
		total++

		chars := utf8.RuneCountInString(m.Text)
		u.Characters += chars
		c.Characters += chars
		totalChars += chars
		if u.Channels == nil {
			u.Channels = make(map[string]int)
		}
		u.Channels[channel]++
		users[m.User] = u
	}
	channels[channel] = c
}

func sortMessages() {
	for _, u := range users {
		if *user != "" && u.Name != *user {
			continue
		}
		sorted = append(sorted, u)
	}

	switch *sortM {
	// TODO add other cases
	case "verbosity":
		sort.Sort(Verbosity{sorted})
	case "characters":
		sort.Sort(Characters{sorted})
	default:
		sort.Sort(Messages{sorted})
	}
}

func display() {
	if false {
		type outUser struct {
			Name        string
			RealName    string
			DisplayName string
			FirstName   string
			LastName    string
		}
		var out []outUser
		for _, u := range users {
			out = append(out, outUser{
				Name:        u.Name,
				RealName:    u.Profile.RealName,
				DisplayName: u.Profile.DisplayName,
				FirstName:   u.Profile.FirstName,
				LastName:    u.Profile.LastName,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "	")
		if err := enc.Encode(out); err != nil {
			log.Fatal("could not encode users to stdout", err)
		}
		return
	}

	if false {
		fmt.Printf("channel\tmessage count\t%% of all messages\tlast message\n")
		//old := time.Now().AddDate(0, -6, 0)
		for n, c := range channels {
			//if c.LastMessage.Before(old) && !c.IsArchived {
			fmt.Printf("%s\t%d\t%.3f\t%s\n", n, c.Count, c.Percent*100, c.LastMessage.Format("2006-01-02"))
			//}
		}
		return
	}

	fmt.Println("Using data dump:", *dir)

	chans := "all channels"
	if *channel != "" {
		chans = "#" + *channel
	}

	var t string
	if from.IsZero() && to.IsZero() {
		t = "for all time"
	} else {
		if !from.IsZero() {
			t = "after " + from.Format("2006-01-02")
		}
		if !to.IsZero() {
			t = "before " + to.Format("2006-01-02")
		}
	}

	fmt.Printf("Showing data for %s %s sorted by %s\n", chans, t, *sortM)

	if *excludeS != "" {
		fmt.Println("Excluding these users:", *excludeS)
	}

	fmt.Println("")

	line := func() {
		col := func(n int) string {
			return strings.Repeat("-", n)
		}
		fmt.Printf(strings.Repeat("+%s", 10)+"+\n", col(5), col(22), col(12), col(12), col(12), col(12), col(12), col(12), col(12), col(54))
	}

	line()
	fmt.Printf("|     | %-20s | %-10s | %-10s | %-10s | %-10s | %-10s | %-10s | %-10s | %-52s |\n", "User", "Messages", "% of total", "Cumulative", "Characters", "% of total", "Cumulative", "Char / Msg", "Favorite Channels")
	line()
	l := *limit
	if l >= len(sorted) {
		l = len(sorted)
	}

	var cumMsg, cumChar float64
	for i, u := range sorted[:l] {
		cumMsg += u.Percent
		cumChar += u.CharPercent
		fmt.Printf("| %3d | %-20s | %10d | %9.3f%% | %9.3f%% | %10d | %9.3f%% | %9.3f%% | %10.2f | %-52s |\n", i+1, u.Name, u.Count, u.Percent*100, cumMsg*100, u.Characters, u.CharPercent*100, cumChar*100, float64(u.Characters)/float64(u.Count), u.FavoriteChannels())
	}
	line()
}

func contains(haystack []string, needle string) bool {
	for i := range haystack {
		if haystack[i] == needle {
			return true
		}
	}
	return false
}
