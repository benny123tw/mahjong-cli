## 1. Wall Construction and Dealing — Add `WallOptions` and `NewWallWithOptions`

- [x] 1.1 In `internal/game/wall_test.go`, add a failing test `TestNewWallWithOptionsAkadoraOnHasOneRedFiveOfEachSuit` that constructs `NewWallWithOptions(42, WallOptions{Akadora: true})`, walks every tile, and asserts exactly one red copy of each of `tile.M5`, `tile.P5`, `tile.S5` (and three plain copies each), per Wall Construction and Dealing.
- [x] 1.2 In `internal/game/wall_test.go`, add a failing test `TestNewWallWithOptionsAkadoraOffHasNoRedTiles` that constructs `NewWallWithOptions(42, WallOptions{Akadora: false})`, walks every tile, and asserts zero tiles with `Red == true` while every ID still has 4 copies, per Wall Construction and Dealing.
- [x] 1.3 In `internal/game/wall_test.go`, add a failing test `TestNewWallAkadoraSubstitutionIsDeterministic` that constructs `NewWallWithOptions(42, WallOptions{Akadora: true})` twice and asserts the two walls are byte-identical tile-by-tile, per Wall Construction and Dealing.
- [x] 1.4 In `internal/game/wall.go`, declare `type WallOptions struct { Akadora bool }` and the `NewWallWithOptions(seed int64, opts WallOptions) *Wall` constructor that performs Substitute Red Fives Before Shuffle: build the 136-tile slice with `Red: false`, then if `opts.Akadora` flip the first encountered `tile.M5`, `tile.P5`, `tile.S5` to `Red: true` before calling `rand.Shuffle`.
- [x] 1.5 In `internal/game/wall.go`, refactor the existing `NewWall(seed int64) *Wall` so it delegates to `NewWallWithOptions(seed, WallOptions{Akadora: true})`, preserving Default-On behaviour.
- [x] 1.6 Run `go test ./internal/game/ -run TestNewWall` and confirm the three new akadora tests plus the existing wall tests all pass.

## 2. Hanchan Match Structure — Thread `MatchOptions` Through `NewMatch` And `NewWithDealer`

- [x] 2.1 In `internal/game/match_test.go`, add a failing test `TestNewMatchDefaultsToAkadoraOn` that constructs `NewMatch(7)`, inspects the active hand's wall, and asserts exactly one red copy of each five, per Hanchan Match Structure.
- [x] 2.2 In `internal/game/match_test.go`, add a failing test `TestNewMatchWithOptionsAkadoraOffPropagatesToEveryHand` that constructs `NewMatchWithOptions(7, MatchOptions{Akadora: false})`, drives `AdvanceFromOutcome` through three rotations (using existing test outcome helpers), and asserts every constructed wall has zero red tiles, per Hanchan Match Structure.
- [x] 2.3 In `internal/game/match.go`, declare `type MatchOptions struct { Akadora bool }`, add `NewMatchWithOptions(seed int64, opts MatchOptions) *Match`, store `opts` on the `Match` struct, and refactor `NewMatch(seed)` to delegate to `NewMatchWithOptions(seed, MatchOptions{Akadora: true})`.
- [x] 2.4 In `internal/game/turn.go`, add `type GameOptions struct { Akadora bool }` and `NewWithDealerOptions(seed int64, dealer Seat, roundWind tile.Tile, opts GameOptions) *Game` that constructs the wall via `NewWallWithOptions(seed, WallOptions{Akadora: opts.Akadora})`. Refactor `NewWithDealer` to delegate to `NewWithDealerOptions` with `GameOptions{Akadora: true}` so existing behaviour is preserved.
- [x] 2.5 In `internal/game/match.go`, update every call site that builds the per-hand `*Game` (initial construction in `NewMatchWithOptions`, and the rotation/renchan path inside `AdvanceFromOutcome`) to call `NewWithDealerOptions(seed, dealer, roundWind, GameOptions{Akadora: m.opts.Akadora})` instead of `NewWithDealer(...)`.
- [x] 2.6 Run `go test ./internal/game/` and confirm the new match-level tests plus all previously passing tests stay green.

## 3. Play Subcommand Launch — `--no-akadora` Opt-Out

- [x] 3.1 In `cmd/play.go`, extend the Play Subcommand Launch wiring by registering a new boolean flag `--no-akadora` (default false) on the play cobra command.
- [x] 3.2 In `cmd/play.go`, in the play `RunE` body, replace `play.NewWithMatch(renderer, game.NewMatch(seed))` with: if `--no-akadora` is true, construct the match via `game.NewMatchWithOptions(seed, game.MatchOptions{Akadora: false})`; otherwise use `game.NewMatch(seed)` (Default-On With `--no-akadora` Opt-Out).
- [x] 3.3 Run `go build ./...` and `mahjong play --help` to confirm the new flag is documented in usage output.

## 4. Verification

- [x] 4.1 Run `go test ./...` from the repository root and confirm all packages pass.
- [x] 4.2 Run `golangci-lint run ./...` and resolve any lint issues introduced by the akadora changes.
- [x] 4.3 Smoke-test by running `mahjong play --seed 42` and confirming that within a few hands a red five (`0p`/`0m`/`0s` glyph) appears in either the wall draws or in a starting hand. Then run `mahjong play --seed 42 --no-akadora` and confirm no red five glyph ever appears.
