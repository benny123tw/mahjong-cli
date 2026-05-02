## Context

The tile model and yaku/dora computation already understand red fives:

- `tile.Tile` has a `Red bool` field; `tile.NewRed(id)` constructs one.
- `tile.Tile.String()` returns `"0p"` / `"0m"` / `"0s"` for red fives.
- `internal/riichi/calc.countDoraHan` increments dora-han for every `t.Red` tile in the winning hand.
- `internal/riichi/score/fu.go` and the yaku detector treat red fives as ordinary 5s for shape purposes.

What's wired but never fires: the wall in `NewWall` builds 4 copies of every tile with `Red: false`, so no red five enters play. To complete the rule set we just need the wall to actually contain red fives and a way to turn the rule off if desired.

The existing `Wall` struct is small and self-contained. The `NewWall(seed int64)` constructor is called from `Game.NewWithDealer`, which is called from `Match.NewMatch` and from per-hand transitions in `Match.AdvanceFromOutcome`. Threading an options struct through is mechanical.

## Goals / Non-Goals

**Goals:**

- Default-on akadora: a fresh `game.NewMatch(seed)` produces a wall with one red 5m, one red 5p, and one red 5s.
- Explicit opt-out: `mahjong play --no-akadora` produces a wall with all-normal fives.
- Backwards compatibility: `NewWall(seed)` still works and defaults to akadora-on (matches modern client expectations).
- Determinism: per-seed wall shuffle remains identical between runs; the red substitution happens BEFORE the shuffle so the position of red fives within the shuffled order is deterministic per seed.
- All existing tests pass — the substitution only affects which physical tile copies have `Red = true`; shape and ID are unchanged.

**Non-Goals:**

- Configurable red-five count (some rule sets use 0, 1, 2, or 3 red fives total). v1 is fixed at 1-of-each-suit (the modern default).
- Red ones, red nines, or other red variants used in some local rules.
- Per-hand toggle: the akadora setting is fixed for the entire hanchan, no mid-match changes.
- Score-screen UI distinction beyond the existing `"0p"` glyph rendering.

## Decisions

### Substitute Red Fives Before Shuffle

In `NewWallWithOptions`, after the tile slice is initialized but before the shuffle:

```go
if opts.Akadora {
    // Find the first occurrence of each five and flip it to red.
    for _, fiveID := range []uint8{tile.M5, tile.P5, tile.S5} {
        for i := range tiles {
            if tiles[i].ID == fiveID && !tiles[i].Red {
                tiles[i].Red = true
                break
            }
        }
    }
}
r.Shuffle(...)
```

The substitution targets the FIRST occurrence of each five-ID; since all four copies have the same `ID` and `Red: false` initially, picking the first is deterministic. After the shuffle, the position of the red five within the wall is determined by the seed.

**Alternative considered:** Replace a random tile after shuffle. Rejected — that introduces a second source of randomness, complicating replay. Substituting before shuffle keeps the seed → wall order mapping clean.

### Default-On With `--no-akadora` Opt-Out

Modern online riichi clients (Tenhou, Mahjong Soul, etc.) default to akadora-on. Mirroring that default matches user expectations and keeps the CLI flag count small (`--seed` and `--no-akadora` instead of `--seed` and `--akadora`).

The legacy `NewWall(seed)` constructor delegates to `NewWallWithOptions(seed, WallOptions{Akadora: true})`. Tests that call `NewWall(seed)` directly will see red fives in the wall, but since the existing test assertions don't check `Red` on individual tiles, behavior is unchanged.

**Alternative considered:** Default-off, opt-in via `--akadora`. Rejected — modern players expect akadora-on; opt-out matches that mental model.

### Thread `MatchOptions` Through `NewMatch` And `NewWithDealer`

`Match.NewMatch` gains a sibling `NewMatchWithOptions(seed int64, opts MatchOptions) *Match`. The legacy `NewMatch(seed)` delegates to `NewMatchWithOptions(seed, MatchOptions{Akadora: true})`. `MatchOptions{Akadora bool}` is stored on the `Match` struct so per-hand transitions in `AdvanceFromOutcome` can re-pass it to `NewWithDealer`.

`Game.NewWithDealer` also gains a sibling `NewWithDealerOptions(seed, dealer, roundWind, gameOpts GameOptions) *Game` where `GameOptions{Akadora bool}` flows into the wall construction. The legacy `NewWithDealer(seed, dealer, roundWind)` delegates with `GameOptions{Akadora: true}`.

**Alternative considered:** Add an `akadora bool` parameter directly to the existing constructors. Rejected — every test that calls `NewWithDealer` would need updating. Wrapping in a struct keeps the v1 entry points stable while allowing the new options field to grow (e.g., red-count tweaks in future changes).

## Risks / Trade-offs

[Risk: existing tests that decompose hands and assume no red fives might break if they rely on specific tile values from a known seed] → Mitigation: the golden-game test (TestGoldenSeed) compares EVENT LOG strings, not tile struct fields. Tile rendering is unchanged for non-fives; for fives, `String()` returns `"0p"` instead of `"5p"` — the golden file would need refreshing if any seed-7 trace happens to draw a red five. Run with `-update` after the change.

[Risk: red fives in melded sets cause a yaku detector to mis-identify the meld type] → Mitigation: the detector already uses `t.ID` exclusively for shape decisions; `t.Red` is only consulted in `countDoraHan`. Existing yaku tests (which use Hand fixtures with no red flags) are unaffected.

[Risk: the deterministic "first 4 copies → first one becomes red" rule means the red five always sits at the same wall position before shuffle, biasing which seed-positions yield red-five-in-hand] → Mitigation: this is exactly the same kind of pre-shuffle invariant that the wall already has (e.g., tile IDs are inserted in iota order before shuffle). The shuffle is the only randomness; per-seed reproducibility is what matters.
