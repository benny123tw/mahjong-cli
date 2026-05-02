## Why

The tile model already supports red fives — `tile.Tile{ID, Red: true}`, `tile.NewRed`, the canonical `"0p"`/`"0m"`/`"0s"` string form, and `calc.countDoraHan` already increments dora-han when it sees `t.Red`. What's missing is the wall: `NewWall` builds 4 copies of every tile with `Red: false` for all of them. As a result, a red five never enters play, and the akadora bonus never fires. Akadora is the most common house rule in modern riichi clients; turning it on completes the playable rule set.

## What Changes

- The wall constructor SHALL substitute one of the four copies of each five (5m, 5p, 5s) with the red variant when akadora is enabled. The default is enabled — modern online riichi defaults to akadora-on; turning it off requires an explicit opt-out.
- A new `NewWallWithOptions(seed int64, opts WallOptions) *Wall` constructor accepts a `WallOptions{Akadora bool}` config. The legacy `NewWall(seed int64)` continues to delegate to `NewWallWithOptions(seed, WallOptions{Akadora: true})` for backward compatibility (old callers automatically get akadora-on, matching modern conventions).
- `play_status` rendering already supports red five glyphs via `tile.String()` returning `"0p"` etc. — no rendering changes needed.
- `cmd/play.go` adds a `--no-akadora` flag (boolean, default false) that disables akadora when set. The seed/flag-feed path threads `WallOptions` through `game.NewMatch`.
- `game.NewMatch` and `game.NewWithDealer` accept the wall options via a new `MatchOptions{Akadora bool}` argument (or default-on if not specified).

## Capabilities

### Modified Capabilities

- `game-loop`: `Wall Construction and Dealing` updated to substitute one red copy per 5-rank tile when akadora is enabled.
- `match-flow`: `Hanchan Match Structure` updated to thread match-level options (akadora on/off) through to per-hand `*Game` construction.
- `play-screen`: `Play Subcommand Launch` updated to add `--no-akadora` opt-out flag.

## Impact

- Modified: `internal/game/wall.go` (new `WallOptions` + `NewWallWithOptions`, red substitution in tile init), `internal/game/wall_test.go` (red presence test + akadora-off test), `internal/game/match.go` (thread `MatchOptions` through `NewMatch` → `NewWithDealer` → `NewWallWithOptions`), `cmd/play.go` (--no-akadora flag).
- New: none — all changes are surgical extensions of existing files.
- Removed: none.
