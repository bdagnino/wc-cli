#!/usr/bin/env bash
# Regenerate the README terminal screenshots in docs/.
#
# Each image shows the command prompt followed by its real output, so readers
# can see exactly what produced it. Color is forced on with CLICOLOR_FORCE so
# the piped output keeps its styling, and charmbracelet/freeze renders the SVG.
#
# Requires: freeze (github.com/charmbracelet/freeze), a built ./bin/wcup.
set -euo pipefail

cd "$(dirname "$0")/.."
go build -o bin/wcup ./cmd/wcup

WCUP="$PWD/bin/wcup"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# Shared freeze styling, matched to the original screenshots.
FREEZE_FLAGS=(
  --window
  --background "#0d1117"
  --font.family "JetBrains Mono"
  --font.size 14
  --border.radius 8
)

# shot <output.svg> <command...> — render a prompt line plus the command's
# colored output to an SVG.
shot() {
  local out="$1"; shift
  local body="$TMP/body.ansi"
  # Prompt line: dim ❯ then the bold command, a blank line, then real output.
  printf '\033[38;5;243m❯ \033[0;1mwcup %s\033[0m\n\n' "$*" >"$body"
  CLICOLOR_FORCE=1 "$WCUP" "$@" >>"$body"
  freeze -x "cat $body" "${FREEZE_FLAGS[@]}" -o "$out"
  echo "wrote $out"
}

shot docs/screenshot-today.svg today
shot docs/screenshot-match.svg match 760414
shot docs/screenshot-standings.svg standings --group A
