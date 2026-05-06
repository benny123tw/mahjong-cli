# mahjong-cli

A Japanese riichi mahjong CLI: an interactive terminal game (`mahjong play`) and
a hand-calculator (`mahjong calc`). Built in Go with Bubble Tea and Lip Gloss.

The play screen is a single-hand TUI against three bots; the calculator
analyzes a 13- or 14-tile hand and reports shanten/machi or full
yaku/fu/score breakdowns. The whole codebase is developed
spec-first via [Spectra](https://github.com/spectra-app/spectra) — specs live
in `openspec/specs/`, archived change history in `openspec/changes/archive/`.

## Install

Pre-built binary (recommended):

```sh
curl -sSfL https://raw.githubusercontent.com/benny123tw/mahjong-cli/main/install.sh \
  | sh -s -- -b /usr/local/bin
```

Pin a specific version by appending the tag:

```sh
curl -sSfL https://raw.githubusercontent.com/benny123tw/mahjong-cli/main/install.sh \
  | sh -s -- -b /usr/local/bin v0.1.0
```

Or use Go (requires Go 1.26+):

```sh
go install github.com/benny123tw/mahjong-cli/cmd/mahjong@latest
```

Or build from source:

```sh
git clone https://github.com/benny123tw/mahjong-cli
cd mahjong-cli
go build -o mahjong ./cmd/mahjong
```

## Usage

### Play

```sh
mahjong play                 # play a single hand against three bots
mahjong play --seed 42       # deterministic shuffle (printed at startup)
mahjong play --ascii         # ASCII tile rendering instead of Unicode glyphs
mahjong play --no-akadora    # disable red fives
mahjong play --demo-end ron  # boot directly into the end-of-hand reveal panel
                             # (useful for inspecting the win/draw display)
```

`--demo-end` accepts: `ron`, `tsumo`, `chankan`, `ryuukyoku`.

### Keymap (during play)

| Key       | Action                                                       |
| --------- | ------------------------------------------------------------ |
| Key       | Action                                                                            |
| --------- | --------------------------------------------------------------------------------- |
| `←` / `→` | Move cursor between tiles in your hand                                            |
| `1` – `9` | Jump cursor to a hand position                                                    |
| `D`       | Discard the tile under the cursor                                                 |
| `R`       | Riichi when on your turn (tenpai + closed hand + ≥ 1000 pts); Ron in call window  |
| `T`       | Tsumo (self-draw win)                                                             |
| `P`       | Pon — call a triplet on an opponent discard                                       |
| `C`       | Chi — call a sequence (kamicha only)                                              |
| `K`       | Kan — currently inert; full UI in a follow-up                                     |
| `Spc`     | Pass the call window                                                              |
| `?`       | Peek — toggle shanten + machi display                                             |
| `Q`       | Quit                                                                              |

The drawn tile is rendered with a one-tile gap separating it from the
sorted main hand; the cursor automatically jumps to it after each draw
so a single `D` press tsumogiri-discards.

### Calc

```sh
mahjong calc 234m234p234s33z11p --tsumo --riichi   # 14-tile, tsumo win
mahjong calc 234m234p234s33z11p                    # 13-tile, shanten + machi
mahjong calc 234m234p234s33z11p --dora 5p          # add a dora indicator
mahjong calc 234m234p234s33z11p --seat E --round E # dealer wind context
```

Tile codes: `1m`..`9m`, `1p`..`9p`, `1s`..`9s` for the suits; `1z`..`7z`
for honors (`1z`–`4z` = E/S/W/N winds, `5z`/`6z`/`7z` = Haku/Hatsu/Chun);
`0m`/`0p`/`0s` for red fives. For a 14-tile hand, the last tile is treated
as the winning tile.

## Yaku coverage

All standard riichi yaku, all kokushi/yakuman variants, and the situational
group (ippatsu, haitei, houtei, rinshan, chankan, double riichi, tenhou,
chiihou). Fu calculation includes the kan-aware path. Akadora is on by
default (toggle with `--no-akadora`).

## Project layout

```
cmd/mahjong/               # binary entrypoint (main package)
cmd/                       # cobra subcommands (play, calc, version, root)
internal/play/             # Bubble Tea TUI: model, render, keys, end panel
internal/game/             # state machine, match progression, payouts, kan flow
internal/riichi/
  ├── tile/                # tile types, parser, sort
  ├── hand/                # decomposition, shanten, machi
  ├── yaku/                # yaku detectors
  ├── score/               # han/fu/base/payout tables
  └── calc/                # top-level Analyze entry point
openspec/
  ├── specs/               # capability specs (play-screen, game-loop, ...)
  └── changes/archive/     # archived change proposals + delta specs
```

## Development

```sh
go test ./...           # all tests
go test -race ./...     # with race detector (CI uses this)
golangci-lint run ./...
```

Spec-driven workflow via Spectra slash commands (in Claude Code):

```
/spectra-discuss   # structure a discussion before coding
/spectra-propose   # create proposal + design + specs + tasks
/spectra-apply     # implement the tasks
/spectra-archive   # archive completed change + sync delta specs
```

CI runs `go test -race ./...` and `golangci-lint run ./...` on every push and
PR to `main`.
