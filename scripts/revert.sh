#!/bin/sh
# Undo scripts/install.sh: SDDM becomes the display manager again. greetd
# stays installed, and /etc/greetd/config.toml stays as it is; it is only
# not started any more. Neither service is stopped or started now, so the
# session on screen stays.

set -eu

[ "$(id -u)" = 0 ] || { echo "joucho: run me with sudo: sudo $0" >&2; exit 1; }

systemctl disable greetd.service >/dev/null 2>&1 || true
if systemctl list-unit-files sddm.service >/dev/null 2>&1; then
    systemctl enable sddm.service >/dev/null
    echo "joucho: greetd disabled, sddm enabled; reboot to see SDDM again"
else
    echo "joucho: sddm is not installed; enable your display manager by hand" >&2
    exit 1
fi
