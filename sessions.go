package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Session is a Wayland session from /usr/share/wayland-sessions.
type Session struct {
	File string   // the file name, "inua.desktop"
	Name string   // its Name, localized if the file has one for the language
	Exec string   // the program to run
	Args []string // the arguments after it
	Icon string   // its Icon, "" if none
}

// Command is what greetd starts.
func (s Session) Command() []string { return append([]string{s.Exec}, s.Args...) }

// defaultSessionDirs is where Wayland sessions live.
var defaultSessionDirs = []string{
	"/usr/local/share/wayland-sessions",
	"/usr/share/wayland-sessions",
}

// readSessions reads the session entries in dirs, in the order given, later
// directories overriding earlier ones of the same name. Entries marked
// Hidden or NoDisplay, or whose TryExec is missing, are left out.
func readSessions(dirs []string, lang string) []Session {
	byFile := map[string]int{} // file name → index in sessions
	var sessions []Session
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".desktop") {
				continue
			}
			s, ok := parseSession(filepath.Join(dir, e.Name()), e.Name(), lang)
			if !ok {
				continue
			}
			if i, dup := byFile[s.File]; dup {
				sessions[i] = s
				continue
			}
			byFile[s.File] = len(sessions)
			sessions = append(sessions, s)
		}
	}
	sort.SliceStable(sessions, func(i, j int) bool { return sessions[i].Name < sessions[j].Name })
	return sessions
}

// parseSession reads one .desktop file's [Desktop Entry] group.
func parseSession(path, file, lang string) (Session, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Session{}, false
	}
	s := Session{File: file}
	inEntry := false
	vals := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			inEntry = line == "[Desktop Entry]"
			continue
		}
		if !inEntry {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		vals[strings.TrimSpace(key)] = strings.TrimSpace(val)
	}
	if vals["Type"] != "" && vals["Type"] != "Application" {
		return Session{}, false
	}
	if vals["Hidden"] == "true" || vals["NoDisplay"] == "true" {
		return Session{}, false
	}
	if vals["Exec"] == "" {
		return Session{}, false
	}
	if try := vals["TryExec"]; try != "" && !haveExec(try) {
		return Session{}, false
	}
	s.Name = localized(vals, "Name", lang)
	if s.Name == "" {
		s.Name = strings.TrimSuffix(file, ".desktop")
	}
	args, err := splitExec(vals["Exec"])
	if err != nil || len(args) == 0 {
		return Session{}, false
	}
	s.Exec, s.Args = args[0], args[1:]
	s.Icon = vals["Icon"]
	return s, true
}

// localized picks key[lang], key[language], else key.
func localized(vals map[string]string, key, lang string) string {
	for _, k := range []string{key + "[" + lang + "]", key + "[" + strings.SplitN(lang, "_", 2)[0] + "]", key} {
		if v := vals[k]; v != "" {
			return v
		}
	}
	return ""
}

// haveExec reports whether a TryExec program exists: an absolute path, or
// a name on PATH.
func haveExec(prog string) bool {
	if strings.ContainsRune(prog, '/') {
		st, err := os.Stat(prog)
		return err == nil && !st.IsDir() && st.Mode()&0o111 != 0
	}
	_, err := exec.LookPath(prog)
	return err == nil
}

// pickSession chooses the session to start on: the configured file, else
// the first.
func pickSession(want string, sessions []Session) Session {
	for _, s := range sessions {
		if s.File == want {
			return s
		}
	}
	return sessions[0]
}

// splitExec splits a desktop entry Exec line: words separated by spaces,
// double quotes group, a backslash escapes the next character, and the
// field codes (%f, %U, …) are dropped.
func splitExec(s string) ([]string, error) {
	var words []string
	var b strings.Builder
	quoted := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s):
			i++
			b.WriteByte(s[i])
		case c == '"':
			quoted = !quoted
		case (c == ' ' || c == '\t') && !quoted:
			if b.Len() > 0 {
				words = append(words, b.String())
				b.Reset()
			}
		default:
			b.WriteByte(c)
		}
	}
	if b.Len() > 0 {
		words = append(words, b.String())
	}
	out := words[:0]
	for _, w := range words {
		if len(w) == 2 && w[0] == '%' && strings.ContainsRune("fFuUdDnNickvm", rune(w[1])) {
			continue
		}
		out = append(out, w)
	}
	if len(out) == 0 {
		return nil, os.ErrInvalid
	}
	return out, nil
}
