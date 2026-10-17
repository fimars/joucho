# Contributing to joucho

Thanks for helping. joucho is small on purpose: it shows greetd's login
screen and looks like [inua](https://github.com/lyricat/inua) while it
does it. Bug reports, fixes, themes, translations and reports from real
machines are all welcome.

## Ways to help

- **Say what broke.** Open an issue with what you did, what happened and
  what you expected. `journalctl -u greetd` (or `/var/log/greetd.log`)
  and the output of `joucho --list` explain most of it. Say whether
  `--demo` shows the problem too — then the bug is ours — or only the
  real login, where greetd's log is the interesting one. Distro, GPU and
  compositor help for anything visual.
- **Test on a real machine.** Most contributors cannot try every distro,
  GPU, keyboard layout or PAM stack; a report that a session starts (or
  does not) is worth a lot. The safe way in and out is in the README:
  `scripts/install.sh`, a reboot, and `scripts/revert.sh` to go back.
- **Themes and translations.** A theme with a token set inua does not
  have, or a language this greeter does not speak yet, is self-contained
  and easy to review.
- **Code.** Say in an issue what you are about to work on, so two people
  do not do the same thing.

## Building and trying

Go 1.26 or newer, and the GTK4 libraries at run time (no cgo and no
headers: puregotk opens `libgtk-4.so` at run time).

```sh
go build -o joucho .
go test ./...
go vet ./...
```

Without greetd, in your running session:

```sh
./joucho --demo                     # the interface, no authentication
./joucho --demo --theme yoru        # another theme
./joucho --demo --theme isekai      # the full composition
./joucho --list                     # users and sessions found
./joucho --render render.png        # isekai with no display at all
```

`--render` is the one to use for anything you nudge in `decor.go`: it
draws the composition one concept unit to one pixel, with the card's
footprint marked, and needs neither a compositor nor a screen.

With a real greetd, install as the README says. To see a change without
rebooting, switch to a TTY (Ctrl+Alt+F3) and run:

```sh
sudo systemctl stop sddm && sudo systemctl start greetd
```

`scripts/revert.sh` puts the display manager back.

## Where things are

| File | What |
| --- | --- |
| `main.go` | the flags, and the pieces wired together |
| `config.go` | `/etc/joucho/config.toml`, defaults and validation |
| `greetd.go` | greetd's IPC: framing, requests, responses (`greetd-ipc(7)`) |
| `ui.go` | window, card, user and session menus, the auth flow, power |
| `style.go` | themes, `text_scale`, font, the ground |
| `decor.go` | the isekai composition, drawn in cairo |
| `render.go` | `--render`, off-screen |
| `cairo.go` | the cairo bindings, with purego |
| `users.go`, `sessions.go` | `/etc/passwd` and `.desktop` sessions |
| `i18n.go` | strings, English and Simplified Chinese |
| `assets/` | built-in themes, `style.css`, the default `config.toml` |
| `docs/` | the README picture |
| `scripts/` | `install.sh` and `revert.sh` |

## What to keep in mind

- **greetd does the sensitive work.** PAM lives in greetd; the greeter
  forwards what was typed over `$GREETD_SOCK` and keeps nothing. Never
  authenticate, read a shadow file, or store a password here.
- **No cgo.** puregotk and purego open the C libraries at run time; keep
  it that way, and keep new C bindings in `cairo.go` where they are
  collected.
- **The look is inua's.** Theme token names, `text_scale` and the field
  texture must stay in step with the shell, so a theme means the same
  thing in both. `NOTICE` records what came from inua.
- **The isekai theme is a fan tribute.** It is drawn after ヰ世界情緒
  (Isekaijoucho) of KAMITSUBAKI STUDIO. Keep the credit and the
  "not affiliated" line that `decor.go` draws and what `NOTICE` states;
  do not add official artwork, logos or fonts, and do not suggest
  affiliation with the artist or the studio.
- **The greeter user sees almost nothing.** It runs as `greetd` and can
  read `/etc/passwd` and `/usr/share/wayland-sessions` — not the user's
  home. A picture, font or theme from a home directory is a bug.
- **Comments explain why.** The house voice is short, lowercase prose:
  what a piece is for and the trap it avoids, not what the line says.

## Code style

`gofmt` and `go vet` are the whole rulebook beyond the above. A few
conventions from the existing code:

- One change per pull request; keep an unrelated fix in its own.
- Standard library first, then the four direct dependencies in `go.mod`;
  a new one needs a reason in the pull request.
- `error` is returned, not logged, when the caller can do something
  about it; `logf` is for what ends up in greetd's log.
- Add a test for parsing and protocol work (see `greetd_test.go` for the
  shape); UI changes are hard to test — say how you checked instead,
  preferably with a `--render` picture.
- Touch `CHANGELOG.md` under `Unreleased` when the change is something
  a user would notice.
- Flags, settings and paths are written down twice: `README.md` and
  `README.zh_CN.md`. Keep the two in step.

## Themes

Built-in themes live in `assets/themes/*.css` and mirror inua's
`ui/themes/`: the same file name and the same `@define-color` token
names, so the greeter and the shell agree. A new built-in theme also
needs its name in `Themes` (`style.go`) and a mention in both READMEs
and `assets/config.toml`.

A theme you make for yourself does not belong in the repository: it goes
in `/etc/joucho/themes/NAME.css`, where it overrides only the tokens it
changes. That override lives beside the config, not inside the program.

## Translations

The greeter's own strings are the `zhCN` map and the `date`/`clock`
formats in `i18n.go`; a new language is a map, a case in `newI18n`, and a
line in the config docs. PAM's messages ("Password:", "Login incorrect")
come from the system and pass through untranslated — do not add them
here, and do not translate log lines, which are read by whoever runs
`journalctl`.

## Commits and pull requests

- A short imperative subject: `Handle an empty session list`, not
  `Fixed stuff`.
- A body that says *why* — the bug, the case that was missing — and how
  it was checked on a machine.
- Reference the issue if there is one.
- Say when a change is deliberately out of scope; it keeps the review
  short.

## Licensing

Contributions are accepted under the Apache License 2.0 (see `LICENSE`).
joucho copies the colour themes, the `scaleText` function and the field
texture drawing from inua, which is also Apache-2.0; `NOTICE` lists them.
If you bring code from somewhere else, say so and put it in `NOTICE` the
same way. Do not contribute code whose license cannot live with
Apache-2.0.
