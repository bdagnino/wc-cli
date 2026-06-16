---
name: world-cup
description: Answer questions about the 2026 FIFA World Cup — live scores, today's matches, schedule/fixtures, results, group standings, teams, match detail, and when a given team plays next — by running the `wcup` CLI. Use whenever the user asks about World Cup matches, scores, who is playing, group tables, a team's next or last game, or match details.
---

# World Cup 2026 via the `wcup` CLI

Answer the user's World Cup question by running `wcup` and reading its output.
Always pass `--json` so you can parse reliably, then summarize the answer in
plain language. Do not dump raw JSON at the user unless they ask for it.

## Finding the binary

Run the first of these that works:

1. `wcup` — if installed (Homebrew / `go install`), it is on `PATH`.
2. `./bin/wcup` — when working inside this repository.
3. `go run .` — from the repo root as a last resort.

If none work, tell the user to install it (`brew install bdagnino/tap/wcup`)
and stop.

## Mapping questions to commands

All commands accept `--json`. Times default to the machine's local timezone;
pass `--tz <zone>` (e.g. `--tz Europe/Madrid`) if the user implies a location.

| The user asks… | Run |
| --- | --- |
| "what's on today", "any games today" | `wcup today --json` |
| "what's live", "current scores" | `wcup live --json` |
| "when does Argentina play next" | `wcup next --team argentina --json` |
| "Brazil's schedule / fixtures" | `wcup schedule --team brazil --json` |
| "results", "what were the scores" | `wcup results --json` (filter with `--team`) |
| "group F table", "standings" | `wcup standings --group F --json` |
| "knockout bracket" | `wcup bracket --json` |
| "everything about a team" | `wcup team <name> --json` |
| "who's in the tournament", team codes | `wcup teams --json` |
| "details / goals for a match" | `wcup match <team-or-id> --json` |

Filters that compose on `schedule`, `results`, `next`:
`--team <name|code>`, `--group <A–L>`, `--date <today|tomorrow|YYYY-MM-DD>`,
`--round <group|r32|r16|qf|sf|final>`, `-n <limit>`.

Team names are fuzzy: `argentina`, `arg`, and `ARG` all work. If a team lookup
returns nothing, run `wcup teams --json` to find the right name/code.

## Answering well

- Lead with the direct answer ("Argentina play Algeria on Tue Jun 16, 21:00 your
  time"), then any useful context (venue, group, current score).
- For "next game" questions, `next --team X` returns the single upcoming match;
  read its `Kick` (kickoff, RFC3339 UTC), `Home`/`Away`, `Venue`, `Group`.
- For live questions, report the score and the `Clock`/`Detail` fields.
- Convert kickoff times to the user's timezone when you know it (`--tz`), and say
  which timezone you used.
- If a command returns an empty list, say so plainly (e.g. "no matches today").
