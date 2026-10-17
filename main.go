package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"codeberg.org/puregotk/puregotk/v4/gio"
	"codeberg.org/puregotk/puregotk/v4/gtk"
)

// logf writes one line to what greetd logs.
func logf(format string, a ...any) { log.Printf(format, a...) }

func main() {
	log.SetFlags(0)
	log.SetPrefix("joucho: ")

	cfgPath := flag.String("config", defaultConfigPath, "config file")
	theme := flag.String("theme", "", "colour theme (overrides the config)")
	uiScale := flag.Int("ui-scale", 0, "UI size, 80 to 200 percent (overrides the config)")
	textScale := flag.Int("text-scale", 0, "text size, 80 to 150 percent (overrides the config)")
	demo := flag.Bool("demo", false, "run without greetd: no real authentication")
	sessionDirs := flag.String("sessions-dir", "", "session directories, colon separated (the system's by default)")
	listFlag := flag.Bool("list", false, "list the users and sessions found, then exit")
	render := flag.String("render", "", "draw the isekai composition to this PNG file and exit")
	flag.Parse()

	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		logf("config: %v; the defaults are in use", err)
	}
	if *theme != "" {
		cfg.Theme = *theme
	}
	if *uiScale >= minUIScale && *uiScale <= maxUIScale {
		cfg.UIScale = *uiScale
	}
	if *textScale >= minTextScale && *textScale <= maxTextScale {
		cfg.TextScale = *textScale
	}

	dirs := defaultSessionDirs
	if *sessionDirs != "" {
		dirs = strings.Split(*sessionDirs, ":")
	}

	g := &greeter{
		cfg:          cfg,
		cfgPath:      *cfgPath,
		demo:         *demo,
		i18n:         newI18n(cfg.Language),
		sessionsDirs: dirs,
	}
	if *render != "" {
		loadCairo()
		if err := g.renderComposite(*render); err != nil {
			logf("render: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	g.users = readUsers()
	if len(g.users) == 0 {
		logf("%s", g.i18n.T("no regular users found"))
		os.Exit(1)
	}
	g.sessions = readSessions(dirs, g.i18n.lang)
	if len(g.sessions) == 0 {
		logf("%s", g.i18n.Tf("no sessions in %s", strings.Join(dirs, ", ")))
		os.Exit(1)
	}
	g.user = pickUser(cfg.User, g.users)
	g.session = pickSession(cfg.Session, g.sessions)
	if *listFlag {
		for _, u := range g.users {
			fmt.Printf("user\t%s\t%s\tuid %d\n", u.Name, u.display(), u.UID)
		}
		for _, s := range g.sessions {
			fmt.Printf("session\t%s\t%s\t%s\n", s.File, s.Name, strings.Join(s.Command(), " "))
		}
		fmt.Printf("default\tuser %s\tsession %s (%s)\n", g.user.Name, g.session.File, g.session.Name)
		os.Exit(0)
	}

	app := gtk.NewApplication("io.github.fimars.Joucho", gio.GApplicationDefaultFlagsValue)
	g.app = app
	g.onActivate = func(gio.Application) { g.activate() }
	app.ConnectActivate(&g.onActivate)
	// GApplication parses the arguments it is given, and they are gone
	// from main already: give it its name alone.
	os.Exit(int(app.Run(1, os.Args[:1])))
}
