---
name: world-cup
description: Answer questions about the 2026 FIFA World Cup — live scores, today's matches, schedule/fixtures, results, group standings, teams, match detail, and when a given team plays next — by running the `wcup` CLI. Use whenever the user asks about World Cup matches, scores, who is playing, group tables, a team's next or last game, or match details.
---

# World Cup 2026 via the `wcup` CLI

> Codex twin of this skill lives at [`AGENTS.md`](../../../AGENTS.md) in the repo
> root. Both describe the same behavior; keep them in sync when you change one.

Answer the user's World Cup question by running `wcup` and reading its output.
Always pass `--json` so you can parse reliably, then summarize the answer in
plain language. Do not dump raw JSON at the user unless they ask for it.

## Finding the binary

Run the first of these that works:

1. `wcup` — if installed (Homebrew / `go install`), it is on `PATH`.
2. `./bin/wcup` — when working inside this repository.
3. `go run ./cmd/wcup` — from the repo root as a last resort.

If none work, tell the user to install it (`brew install bdagnino/tap/wcup`)
and stop.

## Timezones — read the offset, never do UTC math

Kickoff times in the JSON `Kick` field are emitted **in the resolved timezone**,
as RFC3339 **with the local offset** (e.g. `2026-06-17T03:00:00+02:00`), not as
bare UTC. So report the date and clock time straight from `Kick` — do **not**
mentally add or subtract hours from a `...Z` value. That hand-conversion is the
#1 source of wrong answers here.

- By default the zone is the machine's local zone. To be sure which that is,
  check it once (`date +%Z%z` or `readlink /etc/localtime`) and tell the user.
- If the user implies a different location, pass `--tz <zone>` (e.g.
  `--tz America/New_York`); the `Kick` offset then reflects that zone.
- Always state which timezone your answer is in.

## Mapping questions to commands

All commands accept `--json`. Pass `--tz <zone>` when the user implies a
location; otherwise times come back in the machine's local zone.

| The user asks… | Run |
| --- | --- |
| "what's on today", "any games today" | `wcup today --json` |
| "yesterday's / tomorrow's games", "matches on June 25" | `wcup day <yesterday\|tomorrow\|YYYY-MM-DD> --json` |
| "what's live", "current scores" | `wcup live --json` |
| "when does Argentina play next" | `wcup next --team argentina --json` |
| "Brazil's schedule / fixtures" | `wcup schedule --team brazil --json` |
| "results", "what were the scores" | `wcup results --json` (filter with `--team`) |
| "group F table", "standings" | `wcup standings --group F --json` (or `wcup group F --json`) |
| "top scorers", "Golden Boot", "who's scored most" | `wcup scorers --json` (cap with `-n`) |
| "knockout bracket", "the draw" | `wcup bracket --json` (full tree) |
| "how does Brazil reach the final", "Brazil's path/route" | `wcup bracket brazil --json` (one team's path) |
| "everything about a team" | `wcup team <name> --json` |
| "who's in the tournament", team codes | `wcup teams --json` |
| "details / goals for a match" | `wcup match <team-or-id> --json` |

Filters that compose on `schedule`, `results`, `next`:
`--team <name|code>`, `--group <A–L>`, `--date <today|yesterday|tomorrow|YYYY-MM-DD>`,
`--round <group|r32|r16|qf|sf|final>`, `-n <limit>`.

Team names are fuzzy: `argentina`, `arg`, and `ARG` all work. If a team lookup
returns nothing, run `wcup teams --json` to find the right name/code.

## Answering well

- Lead with the direct answer ("Argentina play Algeria on Wed Jun 17, 03:00
  Madrid time"), then any useful context (venue, group, current score).
- For "next game" questions, `next --team X` returns the single upcoming match;
  read its `Kick` (kickoff, already in the resolved zone — see Timezones above),
  `Home`/`Away`, `Venue`, `Group`.
- For live questions, report the score and the `Clock`/`Detail` fields.
- If a command returns an empty list, say so plainly (e.g. "no matches today").
