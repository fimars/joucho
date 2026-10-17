package main

import (
	"os"

	"github.com/BurntSushi/toml"
)

// Config is joucho's settings, read from /etc/joucho/config.toml.
// Every value is the default until the file changes it; a missing file is
// the defaults. Bad values fall back with a warning, as inua's do.
type Config struct {
	Theme        string `toml:"theme"`         // a built-in inua theme, or themes/NAME.css beside the config
	UIScale      int    `toml:"ui_scale"`      // percent, 80 to 200; 0 automatic
	TextScale    int    `toml:"text_scale"`    // percent, 80 to 150
	Language     string `toml:"language"`      // "auto", "en", "zh_CN"
	User         string `toml:"user"`          // login name to start on, "" for the first
	Session      string `toml:"session"`       // .desktop file in the sessions directory to start on
	Wallpaper    string `toml:"wallpaper"`     // picture behind the card, "" for the theme's field texture
	Font         string `toml:"font"`          // window font, "" for the built-in stack (the shell's)
	Power        bool   `toml:"power"`         // show suspend, restart and power off
	ClockSeconds bool   `toml:"clock_seconds"` // show seconds in the clock
}

const (
	defaultConfigPath = "/etc/joucho/config.toml"
	defaultTheme      = "washi"

	minUIScale   = 80
	maxUIScale   = 200
	minTextScale = 80
	maxTextScale = 150
)

// Default returns the defaults, which the file changes.
func defaultConfig() Config {
	return Config{
		Theme:     defaultTheme,
		UIScale:   0,
		TextScale: 100,
		Language:  "auto",
		Session:   "inua.desktop",
		Power:     true,
	}
}

// loadConfig reads path over the defaults. A missing file is not an error.
func loadConfig(path string) (Config, error) {
	c := defaultConfig()
	if _, err := toml.DecodeFile(path, &c); err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return c, err
	}
	if c.UIScale != 0 && (c.UIScale < minUIScale || c.UIScale > maxUIScale) {
		c.UIScale = 0
	}
	if c.TextScale < minTextScale || c.TextScale > maxTextScale {
		c.TextScale = 100
	}
	return c, nil
}
