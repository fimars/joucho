# Changelog

Notable changes to joucho are recorded here.

The format is [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project intends to use [Semantic Versioning](https://semver.org/).
Nothing has been tagged yet: the first commit is this MVP, and everything
below stays under `Unreleased` until the first release moves it out
unchanged.

## [Unreleased]

### Added

- The greetd conversation over `$GREETD_SOCK`: length-prefixed JSON as
  `greetd-ipc(7)` documents — `create_session`, answers to PAM's
  prompts, `start_session`, and cancellation when the user or session
  changes. PAM and starting the session stay greetd's work.
- The sign-in card: a clock (seconds optional) and date in the chosen
  language, a user picker (regular users from `/etc/passwd`, the one who
  logged in last this boot preselected), a session picker (Wayland
  sessions with localized names; `Hidden`, `NoDisplay` and a missing
  `TryExec` honoured), password entry, greetd's messages, and Sign in,
  which can be retried after a failed attempt.
- Suspend, restart and shut down buttons through
  `org.freedesktop.login1`, and a setting to hide them.
- The eight built-in [inua](https://github.com/lyricat/inua) colour
  themes, `text_scale` (font sizes only, as the shell's), a configurable
  window font with a warning when it is missing for the greeter user,
  and the theme's field texture — or a picture — as the ground.
- The `isekai` theme's composition, drawn with cairo in the concept's
  own units: paper grain, the night panel, the ヰ, vines and the rust
  thread, the eye-flower, vertical writing, micro labels and the
  palette, with the card set in the concept's own type. A fan tribute
  to ヰ世界情緒; the credit and the fan-concept note are in `NOTICE`.
- Settings in `/etc/joucho/config.toml`, defaults from
  `assets/config.toml`, and custom themes at
  `/etc/joucho/themes/NAME.css`.
- English and Simplified Chinese, `language = "auto"` following `LANG`,
  `LC_ALL` and `LC_MESSAGES`, and a Chinese `README.zh_CN.md` beside the
  English one.
- `--demo` (the whole interface without greetd), `--list` (users and
  sessions found), and `--render` (the isekai composition to a PNG,
  with no display).
- `scripts/install.sh`, which installs greetd and cage, the program,
  the settings and greetd's own config, and switches the display
  manager over; and `scripts/revert.sh` to put the display manager
  back.
- Tests for the greetd wire format and desktop `Exec` parsing.
- Apache-2.0 licensing (`LICENSE`), with the attribution for the parts
  taken from inua in `NOTICE`.

### Fixed

- The suspend button asked for `system-suspend-symbolic`, which
  adwaita-icon-theme 50 no longer carries, so it showed the
  missing-image icon. It now uses `weather-clear-night-symbolic`, the
  same moon the inua icons draw.
