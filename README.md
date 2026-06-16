# wcup

Follow the **2026 FIFA World Cup** from your terminal — live scores, schedule,
standings, and teams. No account, no API key, no browser.

<p align="center">
  <img src="docs/screenshot-today.svg" alt="wcup today" width="640">
</p>

Data comes straight from ESPN's public soccer API — completely free and
unauthenticated.

## Install

### Homebrew (macOS / Linux)

```sh
brew install bdagnino/tap/wcup
```

### Go

```sh
go install github.com/bdagnino/wcup@latest
```

### Binaries

Grab a prebuilt binary for your platform from the
[releases page](https://github.com/bdagnino/wcup/releases).

## Usage

Run `wcup` with no arguments for a smart summary — what's **live now**, else
**today's** matches, else the **next** ones up.

| Command | What it shows |
| --- | --- |
| `wcup` | Smart summary (live → today → next) |
| `wcup today` | Today's matches, with live scores |
| `wcup live` | Only matches in progress right now |
| `wcup next` | The next upcoming match |
| `wcup schedule` | Upcoming fixtures across the tournament |
| `wcup results` | Recently finished matches |
| `wcup standings` | Group tables |
| `wcup bracket` | Knockout bracket |
| `wcup match` | Detail for a live / featured match |
| `wcup team <name>` | A team's fixtures, results, and group |
| `wcup teams` | List all teams (or search for one) |

### Match detail

`wcup match <id|team>` shows the scoreline plus a goal/card/substitution
timeline and game info. With no argument it picks the live or featured match.

<p align="center">
  <img src="docs/screenshot-match.svg" alt="wcup match detail" width="560">
</p>

### Standings

<p align="center">
  <img src="docs/screenshot-standings.svg" alt="wcup standings" width="520">
</p>

### Filters

`schedule`, `results`, and `next` accept:

```sh
wcup schedule --team brazil          # by team (name or 3-letter code)
wcup schedule --group F              # by group
wcup schedule --date tomorrow        # today | tomorrow | YYYY-MM-DD
wcup schedule --round qf             # group | r32 | r16 | qf | sf | final
wcup next --team ARG -n 3            # limit how many show
```

Don't know a team's name or code? Search for it:

```sh
$ wcup teams cong
  🇨🇩 COD Congo DR
```

### Live mode

Add `--watch` (`-w`) to refresh in place every 30 seconds:

```sh
wcup live --watch
```

### Global flags

| Flag | Effect |
| --- | --- |
| `--tz <zone>` | Show kickoff times in a timezone (e.g. `Europe/Madrid`); defaults to local |
| `--json` | Machine-readable output for scripting |
| `--no-color` | Disable colors (also respects `NO_COLOR`) |

## How it works

`wcup` reads ESPN's public, unauthenticated soccer endpoints. The data source
sits behind a small `Provider` interface, so additional backends could be added
without touching the command layer.

## Development

```sh
go build -o bin/wcup .   # build
go vet ./...             # lint
./bin/wcup today         # run
```

Requires Go 1.23+.

## License

[MIT](LICENSE)
