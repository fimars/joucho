#!/bin/sh
# joucho's installer: the greeter, its settings, and greetd to run it.
# Run it as root from the repository:
#
#   sudo scripts/install.sh
#
# It installs greetd and cage (dnf), /usr/local/bin/joucho, the
# settings, and /etc/greetd/config.toml; it disables the display manager in
# use and enables greetd, but starts and stops neither, so the session on
# screen stays. Reboot to see the login screen, or switch from a TTY (see
# the note at the end). --no-dnf keeps it from installing packages.

set -eu

usage() {
    echo "usage: sudo $0 [--no-dnf]" >&2
    exit 2
}

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
no_dnf=false
for arg in "$@"; do
    case "$arg" in
        --no-dnf) no_dnf=true ;;
        -h|--help) usage ;;
        *) usage ;;
    esac
done

[ "$(id -u)" = 0 ] || { echo "joucho: run me with sudo: sudo $0" >&2; exit 1; }

if ! $no_dnf; then
    if ! command -v greetd >/dev/null 2>&1 || ! command -v cage >/dev/null 2>&1; then
        echo "joucho: installing greetd and cage"
        dnf install -y greetd cage
    fi
fi
command -v greetd >/dev/null 2>&1 || { echo "joucho: greetd is not installed" >&2; exit 1; }
command -v cage >/dev/null 2>&1 || { echo "joucho: cage is not installed" >&2; exit 1; }

# The program. Built already, or built here.
bin=$root/joucho
if [ ! -x "$bin" ]; then
    if command -v go >/dev/null 2>&1; then
        echo "joucho: building"
        (cd "$root" && go build -o joucho .)
    else
        echo "joucho: $bin is missing; run 'go build -o joucho .' as your user first" >&2
        exit 1
    fi
fi
install -Dm755 "$bin" /usr/local/bin/joucho

# Settings, and a place for themes of your own.
install -d -m755 /etc/joucho/themes
if [ ! -e /etc/joucho/config.toml ]; then
    install -m644 "$root/assets/config.toml" /etc/joucho/config.toml
    echo "joucho: wrote /etc/joucho/config.toml"
fi

# The greeter's account is the greetd package's (greetd on Fedora).
user=
for name in greetd _greetd greeter; do
    if getent passwd "$name" >/dev/null 2>&1; then
        user=$name
        break
    fi
done
[ -n "$user" ] || { echo "joucho: greetd's account is missing; is greetd installed?" >&2; exit 1; }

# greetd runs cage, which shows the greeter full screen. VT 1 is Fedora's
# greetd default; its unit already conflicts with getty@tty1.
install -d -m755 /etc/greetd
config=/etc/greetd/config.toml
staged=$(mktemp)
trap 'rm -f "$staged"' EXIT
cat >"$staged" <<EOF
# Written by joucho's installer.
# greetd starts cage, which shows joucho full screen.
# See greetd(5) and greetd-ipc(7).

[terminal]
vt = 1

[default_session]
command = "$(command -v cage) -s -- /usr/local/bin/joucho"
user = "$user"
EOF
if [ -f "$config" ] && cmp -s "$staged" "$config"; then
    echo "joucho: $config is already in place"
else
    if [ -f "$config" ]; then
        n=1
        while [ -e "$config.~$n~" ]; do
            n=$((n + 1))
        done
        cp -p "$config" "$config.~$n~"
        echo "joucho: kept the old $config as $config.~$n~"
    fi
    install -m644 "$staged" "$config"
    echo "joucho: wrote $config: cage -s, greeter user $user, VT 1"
fi

# The display manager: greetd in, the others out. Neither is started nor
# stopped now, so the session on screen stays as it is.
for dm in sddm.service gdm.service lightdm.service lxdm.service ly.service; do
    if systemctl list-unit-files "$dm" >/dev/null 2>&1 && systemctl is-enabled "$dm" >/dev/null 2>&1; then
        systemctl disable "$dm" >/dev/null
        echo "joucho: disabled $dm"
    fi
done
systemctl enable greetd.service >/dev/null 2>&1
echo "joucho: enabled greetd.service for the next boot"

cat <<'EOF'

Done. The session on screen is untouched.

To see the login screen, reboot; or, from a TTY (Ctrl+Alt+F3), stop the
display manager and start greetd for the rest of this boot:

  sudo systemctl stop sddm && sudo systemctl start greetd

Settings:      /etc/joucho/config.toml
Themes:        /etc/joucho/themes/NAME.css
Preview:       joucho --demo            (a window in your session)
Back to SDDM:  sudo scripts/revert.sh
EOF
