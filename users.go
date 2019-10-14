package main

import (
	"sort"
	"strings"
)

// User is a slack user
type User struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Profile struct {
		RealName    string `json:"real_name"`
		DisplayName string `json:"display_name"`
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
	} `json:"profile"`

	Count       int
	Percent     float64
	Characters  int
	CharPercent float64
	Channels    map[string]int
}

// FavoriteChannels gives the four channels where this user has talked most
func (u User) FavoriteChannels() string {
	pl := make(PairList, len(u.Channels))
	i := 0
	for k, v := range u.Channels {
		pl[i] = Pair{k, v}
		i++
	}
	sort.Sort(sort.Reverse(pl))

	var fav []string
	for i := 0; i < 4 && i < len(pl); i++ {
		fav = append(fav, "#"+pl[i].Key)
	}
	return strings.Join(fav, ", ")
}

// Users is a slice of Users with additional methods added for sorting
type Users []User

// Len implements sort.Interface
func (s Users) Len() int { return len(s) }

// Swap implements sort.Interface
func (s Users) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// Messages sorts a Users by number of messages sent descending.
type Messages struct{ Users }

// Less implements sort.Interface
func (s Messages) Less(i, j int) bool { return s.Users[i].Count > s.Users[j].Count }

// Characters sorts a Users by number of characters sent descending.
type Characters struct{ Users }

// Less implements sort.Interface
func (s Characters) Less(i, j int) bool { return s.Users[i].Characters > s.Users[j].Characters }

// Verbosity sorts a Users by average number of characters per message
type Verbosity struct{ Users }

func avg(u User) float64 {
	if u.Count == 0 {
		return 0
	}
	return float64(u.Characters) / float64(u.Count)
}

// Less implements sort.Interface
func (s Verbosity) Less(i, j int) bool {
	return avg(s.Users[i]) > avg(s.Users[j])
}
