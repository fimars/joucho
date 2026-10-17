package main

import (
	"os"
	"sort"
	"strconv"
	"strings"
)

// User is a regular user of this machine, from /etc/passwd.
type User struct {
	Name  string // login name
	Real  string // the full name, "" if none
	UID   int
	Shell string
}

// display is the name shown on the card: the full name if there is one.
func (u User) display() string {
	if u.Real != "" && u.Real != u.Name {
		return u.Real
	}
	return u.Name
}

// noLogin shells are not offered.
var noLogin = map[string]bool{
	"": true, "/bin/false": true, "/usr/bin/false": true,
	"/sbin/nologin": true, "/usr/sbin/nologin": true,
	"/sbin/halt": true, "/sbin/shutdown": true, "/sbin/suspend": true,
	"/bin/nologin": true, "/bin/halt": true, "/bin/shutdown": true,
}

// readUsers lists the regular users: uid 1000 to 59999, with a shell one
// can log in to. They come in uid order, which is stable and, on most
// machines, oldest account first.
func readUsers() []User {
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return nil
	}
	var users []User
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Split(line, ":")
		if len(f) < 7 {
			continue
		}
		uid, err := strconv.Atoi(f[2])
		if err != nil || uid < 1000 || uid > 59999 {
			continue
		}
		if noLogin[f[6]] {
			continue
		}
		real, _, _ := strings.Cut(f[4], ",")
		users = append(users, User{Name: f[0], Real: real, UID: uid, Shell: f[6]})
	}
	sort.Slice(users, func(i, j int) bool { return users[i].UID < users[j].UID })
	return users
}

// pickUser chooses the user to start on: the configured one, else the last
// one to have a session here, else the first. lastUser is read from
// /run/user/<uid> (the users who logged in this boot).
func pickUser(want string, users []User) User {
	for _, u := range users {
		if u.Name == want {
			return u
		}
	}
	best := users[0]
	bestTime := int64(-1)
	for _, u := range users {
		if st, err := os.Stat("/run/user/" + strconv.Itoa(u.UID)); err == nil {
			if t := st.ModTime().Unix(); t > bestTime {
				best, bestTime = u, t
			}
		}
	}
	return best
}
