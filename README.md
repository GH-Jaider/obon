# obon

**Your agents summon dev servers and never dismiss them. obon shows you every
spirit still lingering on your ports — and sends them off.**

The name comes from Obon (お盆), the Japanese festival in which the spirits of
ancestors return home for a few days and, at its close, are guided away with
paper lanterns floated down the river — tōrō nagashi. That is exactly what
this tool does: Claude Code, Codex, Cursor and friends light dev servers that
keep burning for days. `obon` shows you every port still occupied, who lit it,
and lets you set it free.

```
obon                                                       43 spirits on the river · 1 lit by agents
  PORT ↑ PROTO   PROCESS                               PID     ORIGIN    UPTIME   CWD

▸·3722  UDP     rapportd                              47503   manual    14d13h   /
 ·4173  TCP     node                                  11383   manual    5d11h    /private/tmp/clau…
 ·4321  TCP     node                                  93986   manual    13d14h   /Users/jai/Projec…
 ·5173  TCP     node                                  88409   claude    1d13h    /Users/jai/Projec…

42 shown  /usr/libexec/rapportd
j/k move · space select · x send off · enter detail · / filter · s sort · r rescan · ? help · q quit
```

*(screenshots placeholder — replace with a capture of your terminal)*

## Why

AI coding agents start `vite`, `next`, `serve`, `python3 -m http.server`…
and when the session ends, the servers stay. Days later your port 3000 is
taken by a ghost you can't name. `obon list` answers *what* is on each port;
the TUI answers *who summoned it* (parent process chain up to launchd/init)
and sends it off when you're ready.

## Install

```sh
go install github.com/GH-Jaider/obon@latest
# or
brew tap GH-Jaider/homebrew-obon
brew install obon
```

Requires macOS or Linux; Windows builds best-effort.

## Usage

```sh
obon                       # the lantern board (TUI)
obon list                  # one-shot table
obon list --json           # machine-readable
obon kill 3000 8123        # send off by port or pid (asks first; -y skips)
obon clean --older-than 2h # decant listeners younger than 2h
```

`clean` is named after tōrō nagashi: anything still burning that was lit less
than `<duration>` ago is guided downstream.

## TUI keys

| key | action |
|---|---|
| `j`/`k`, arrows | move |
| `g`/`G`, pgup/pgdn | top/bottom, page |
| `enter` | detail panel: full command, cwd, parent chain tree, all sockets |
| `space` | toggle selection |
| `a` / `A` | select all visible / clear selection |
| `x` | send off selection (or cursor row) |
| `/` | filter — port, process, cmdline, cwd, origin, user (regex if valid) |
| `s` / `S` | cycle sort column / flip direction |
| `r` | rescan now |
| `esc` | clear selection, then filter |
| `?` | full help |
| `q` | quit |

The board refreshes every 2s (configurable). Your cursor follows its row by
identity across refreshes; rows that just appeared hold a brief warm highlight
before settling; rows that disappeared drift away dimmed for one cycle before
being removed.

A `*` after PROTO means the socket is bound to all interfaces (`0.0.0.0`/`::`),
not just loopback. The ORIGIN column names the agent ancestor that spawned the
process (`claude`, `codex`, `code`, …) or `manual`.

## Configuration

`~/.config/obon/config.json`:

```json
{
  "interval_s": 2,
  "agents": ["claude", "codex", "cursor", "code", "aider", "gemini", "tmux", "ssh"]
}
```

## Design notes

- **Data**: `internal/scan` uses [gopsutil](https://github.com/shirou/gopsutil)
  only — no shelling out to `lsof`/`ss`. On Linux it reads `/proc` natively;
  on macOS gopsutil itself delegates to `lsof` under the hood (documented
  upstream behaviour), which means PID resolution works unprivileged.
- **Send-off**: SIGTERM first, SIGKILL after a grace period, then a live probe
  confirms the port actually answered.
- **Privacy**: the TUI never displays environment variables.
- **Metaphor budget**: the lantern lives in the accent colour, the appear/
  disappear transitions, the confirmation copy and this README. Column headers
  say PORT, not SPIRITS.

## Development

```sh
make check test   # vet + staticcheck + tests
make build        # ./bin/obon
make release      # goreleaser snapshot
```

Releases are cut with [goreleaser](https://goreleaser.com); the Homebrew tap
formula targets `GH-Jaider/homebrew-obon`.

## License

MIT
