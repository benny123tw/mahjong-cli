# kan-flow Specification

## Purpose

TBD - created by archiving change 'add-kan-support'. Update Purpose after archive.

## Requirements

### Requirement: Ankan Declaration

The system SHALL accept `InputDeclareAnkan{TileID uint8}` from the active seat in `StateAwaitingDiscard`. The input is valid only when the seat's concealed hand contains exactly 4 tiles with the matching `ID`. On success, the system SHALL: (1) remove the four matching tiles from the concealed hand, (2) append a `Meld{Kind: MeldKan, KanKind: KanAnkan, Tiles: [4]tile.Tile{...}}` to the seat's open melds, (3) reveal the next kan-dora indicator (appended to `g.doraIndicators`), (4) draw a rinshan replacement tile from the dead wall via `Wall.RinshanDraw()` and append it to the seat's hand, (5) set the seat's `lastDrawWasRinshan` flag to true, (6) keep the active seat unchanged in `StateAwaitingDiscard{Player: seat}`. On failure (no 4-of-a-kind, or rinshan draw exhausted at 4 kans), the system SHALL return `ErrIllegalKan` and leave game state unchanged.

#### Scenario: Ankan succeeds when hand has four matching tiles

- **GIVEN** a seat in `StateAwaitingDiscard` with concealed hand containing four `5p` tiles
- **WHEN** the seat submits `InputDeclareAnkan{TileID: tile.P5}`
- **THEN** the four `5p` tiles are removed from the concealed hand
- **AND** a `Meld{Kind: MeldKan, KanKind: KanAnkan}` for `5p` is appended to the seat's melds
- **AND** the seat's hand has 14 tiles (10 remaining + 1 rinshan replacement... wait, 14 - 4 + 1 = 11; but the seat had 14 tiles before kan so the post-kan count is 14 - 4 + 1 = 11. The state remains AwaitingDiscard so the seat must discard or tsumo on their 11-tile hand)
- **AND** `Game.DoraIndicators()` length increased by 1
- **AND** the state is `StateAwaitingDiscard{Player: <same seat>}`

#### Scenario: Ankan rejected when hand has only three matching tiles

- **GIVEN** a seat in `StateAwaitingDiscard` with concealed hand containing three `5p` tiles (and a 14th tile that is not `5p`)
- **WHEN** the seat submits `InputDeclareAnkan{TileID: tile.P5}`
- **THEN** the engine returns `ErrIllegalKan`
- **AND** game state is unchanged


<!-- @trace
source: add-kan-support
updated: 2026-05-02
code:
  - internal/game/turn.go
  - internal/game/call.go
  - internal/game/match.go
  - internal/game/bot.go
  - testdata/game/golden/seed-42.json
  - internal/play/play.go
  - internal/game/kan.go
  - cmd/play.go
  - internal/game/wall.go
  - internal/game/state.go
  - internal/play/kan_keys.go
  - internal/game/payout.go
tests:
  - internal/game/payout_test.go
  - internal/play/play_test.go
  - internal/game/turn_test.go
  - internal/game/wall_test.go
  - internal/game/bot_test.go
  - internal/game/kan_test.go
  - internal/game/match_test.go
  - internal/play/kan_keys_test.go
  - internal/game/furiten_test.go
-->

---
### Requirement: Minkan Declaration

The system SHALL accept `Claim{Kind: ClaimKan}` in `StateAwaitingClaims` from any non-discarder seat whose hand contains exactly 3 tiles matching the discard's `ID`. When `ResolveClaims` selects a kan claim as the winner (kan beats pon, ties broken by closest-to-discarder going right-around-the-table), the system SHALL: (1) remove the three matching tiles from the claimant's concealed hand, (2) build a `Meld{Kind: MeldKan, KanKind: KanMinkan, Tiles: [4]tile.Tile{discard, +3 from hand}, From: discarder}`, (3) pop the discarded tile from the discarder's pond (it's been called), (4) close every open ippatsu window (any call breaks ippatsu), (5) reveal the next kan-dora indicator, (6) draw a rinshan replacement tile, (7) set the claimant's `lastDrawWasRinshan = true`, (8) transition to `StateAwaitingDiscard{Player: claimant}`. The claimant's hand now has the meld plus their concealed remainder + rinshan replacement.

#### Scenario: Minkan claim wins over pon claim on the same discard

- **GIVEN** a seat with three `5p` in hand, another seat with two `5p` in hand
- **WHEN** the East player discards `5p` and both seats submit claims (one `ClaimPon`, one `ClaimKan`)
- **THEN** `ResolveClaims` picks the kan
- **AND** the kan-claimant's hand has a new `MeldKan` for `5p`
- **AND** the engine drew a rinshan replacement tile
- **AND** the state is `StateAwaitingDiscard{Player: <kan-claimant>}`


<!-- @trace
source: add-kan-support
updated: 2026-05-02
code:
  - internal/game/turn.go
  - internal/game/call.go
  - internal/game/match.go
  - internal/game/bot.go
  - testdata/game/golden/seed-42.json
  - internal/play/play.go
  - internal/game/kan.go
  - cmd/play.go
  - internal/game/wall.go
  - internal/game/state.go
  - internal/play/kan_keys.go
  - internal/game/payout.go
tests:
  - internal/game/payout_test.go
  - internal/play/play_test.go
  - internal/game/turn_test.go
  - internal/game/wall_test.go
  - internal/game/bot_test.go
  - internal/game/kan_test.go
  - internal/game/match_test.go
  - internal/play/kan_keys_test.go
  - internal/game/furiten_test.go
-->

---
### Requirement: Shouminkan Declaration And Chankan Window

The system SHALL accept `InputDeclareShouminkan{TileID uint8}` from the active seat in `StateAwaitingDiscard` when the seat has an existing open `Meld{Kind: MeldPon}` for that tile ID AND the seat's concealed hand contains a tile with the matching ID. On declaration, the engine SHALL transition to a new `StateAwaitingChankan{UpgradeTile, Declarer}` and accept `InputResolveClaims` from any non-declarer seat. Only `ClaimRon` is honored in this state (pon/chi/kan claims are ignored). If a valid ron claim is submitted (yaku-bearing winning shape, not furiten), the round terminates as `OutcomeRon{Winner, Loser: declarer, Tile: UpgradeTile}` and the engine SHALL pass `Chankan = true` to `calc.Analyze` via `Game.contextForWin`. If no ron claim is submitted (or all rons are rejected), the engine SHALL: (1) upgrade the existing pon meld in place (set `KanKind: KanShouminkan`, append the 4th tile), (2) remove the 4th tile from the declarer's concealed hand, (3) reveal the next kan-dora indicator, (4) draw a rinshan replacement tile and set `lastDrawWasRinshan = true`, (5) transition back to `StateAwaitingDiscard{Player: declarer}`.

#### Scenario: Shouminkan opens chankan window and completes when no ron

- **GIVEN** a seat with an open `MeldPon` for `5p` and a `5p` tile in their concealed hand, in `StateAwaitingDiscard`
- **WHEN** the seat submits `InputDeclareShouminkan{TileID: tile.P5}`
- **THEN** the state transitions to `StateAwaitingChankan{UpgradeTile: 5p, Declarer: <seat>}`
- **AND** when no opponent submits ron and `InputResolveClaims{Claims: nil}` is processed
- **THEN** the pon meld is upgraded to `KanShouminkan`
- **AND** the rinshan replacement tile is drawn
- **AND** the state is `StateAwaitingDiscard{Player: <declarer>}`

#### Scenario: Chankan ron pre-empts shouminkan

- **GIVEN** a shouminkan declaration is pending in `StateAwaitingChankan{UpgradeTile: 5p}`
- **AND** an opposing seat is tenpai on `5p` and not in furiten
- **WHEN** the opposing seat submits `Claim{Kind: ClaimRon}`
- **THEN** the round terminates with `OutcomeRon{Winner: <opposing seat>, Loser: <declarer>, Tile: 5p}`
- **AND** the winning context has `Chankan = true`
- **AND** the upgraded pon meld is NOT modified


<!-- @trace
source: add-kan-support
updated: 2026-05-02
code:
  - internal/game/turn.go
  - internal/game/call.go
  - internal/game/match.go
  - internal/game/bot.go
  - testdata/game/golden/seed-42.json
  - internal/play/play.go
  - internal/game/kan.go
  - cmd/play.go
  - internal/game/wall.go
  - internal/game/state.go
  - internal/play/kan_keys.go
  - internal/game/payout.go
tests:
  - internal/game/payout_test.go
  - internal/play/play_test.go
  - internal/game/turn_test.go
  - internal/game/wall_test.go
  - internal/game/bot_test.go
  - internal/game/kan_test.go
  - internal/game/match_test.go
  - internal/play/kan_keys_test.go
  - internal/game/furiten_test.go
-->

---
### Requirement: Wall Rinshan Replacement Draw

The system SHALL expose `Wall.RinshanDraw() (tile.Tile, bool)` returning the next replacement tile from the dead wall. The tile is drawn from a fixed slot in the dead-wall reserve (the four positions at indices `[deadWallStart..deadWallStart+3]`, indexed by kan count). After 4 successful kan in a single round, `RinshanDraw` SHALL return `false`. The function SHALL NOT decrement the live wall's draw index — `Wall.LiveRemaining()` returns the same value before and after a rinshan draw.

#### Scenario: Rinshan draw does not consume from live wall

- **GIVEN** a fresh wall with `LiveRemaining() == 70`
- **WHEN** `Wall.RinshanDraw()` is called once
- **THEN** the returned tile is non-zero and `ok == true`
- **AND** `Wall.LiveRemaining()` is still 70

#### Scenario: Rinshan exhausts after 4 kans

- **WHEN** `Wall.RinshanDraw()` is called 4 times successfully
- **AND** a 5th call is attempted
- **THEN** the 5th call returns `(zero, false)`


<!-- @trace
source: add-kan-support
updated: 2026-05-02
code:
  - internal/game/turn.go
  - internal/game/call.go
  - internal/game/match.go
  - internal/game/bot.go
  - testdata/game/golden/seed-42.json
  - internal/play/play.go
  - internal/game/kan.go
  - cmd/play.go
  - internal/game/wall.go
  - internal/game/state.go
  - internal/play/kan_keys.go
  - internal/game/payout.go
tests:
  - internal/game/payout_test.go
  - internal/play/play_test.go
  - internal/game/turn_test.go
  - internal/game/wall_test.go
  - internal/game/bot_test.go
  - internal/game/kan_test.go
  - internal/game/match_test.go
  - internal/play/kan_keys_test.go
  - internal/game/furiten_test.go
-->

---
### Requirement: Kan Dora Indicator Reveal

After every successful kan (ankan, minkan, or completed shouminkan), the system SHALL reveal one additional dora indicator from the dead wall and append it to `Game.DoraIndicators()`. The reveal happens immediately after the rinshan replacement draw, before the post-kan `StateAwaitingDiscard` is entered. Subsequent agari calls that include `Game.contextForWin` SHALL see the augmented dora list when computing dora han.

#### Scenario: Ankan reveals an additional dora indicator

- **GIVEN** a fresh game with `len(Game.DoraIndicators()) == 1`
- **WHEN** the active seat declares ankan successfully
- **THEN** `len(Game.DoraIndicators()) == 2`


<!-- @trace
source: add-kan-support
updated: 2026-05-02
code:
  - internal/game/turn.go
  - internal/game/call.go
  - internal/game/match.go
  - internal/game/bot.go
  - testdata/game/golden/seed-42.json
  - internal/play/play.go
  - internal/game/kan.go
  - cmd/play.go
  - internal/game/wall.go
  - internal/game/state.go
  - internal/play/kan_keys.go
  - internal/game/payout.go
tests:
  - internal/game/payout_test.go
  - internal/play/play_test.go
  - internal/game/turn_test.go
  - internal/game/wall_test.go
  - internal/game/bot_test.go
  - internal/game/kan_test.go
  - internal/game/match_test.go
  - internal/play/kan_keys_test.go
  - internal/game/furiten_test.go
-->

---
### Requirement: Rinshan And Chankan Context Flags

When the engine builds `calc.Context` via `Game.contextForWin(winner, isTsumo)`:

- For tsumo wins (`isTsumo = true`), `Rinshan` SHALL be set to `true` if and only if the winner's most recent intake was via `Wall.RinshanDraw()` (tracked by per-seat `lastDrawWasRinshan` flag, cleared on next discard or call).
- For ron wins (`isTsumo = false`), `Chankan` SHALL be set to `true` if and only if the round terminated from the chankan window (`StateAwaitingChankan` resolution path).

The yaku detector already evaluates `yaku.Rinshan` and `yaku.Chankan` based on these flags; this requirement only ensures the flags are populated correctly by the engine.

#### Scenario: Tsumo on rinshan replacement triggers Rinshan flag

- **GIVEN** the active seat just declared a successful ankan and drew a rinshan replacement tile that completes their hand
- **WHEN** the seat submits `InputDeclareTsumo`
- **THEN** the round terminates with `OutcomeTsumo`
- **AND** the winning context has `Rinshan = true`
- **AND** the yaku list includes "rinshan kaihou"

<!-- @trace
source: add-kan-support
updated: 2026-05-02
code:
  - internal/game/turn.go
  - internal/game/call.go
  - internal/game/match.go
  - internal/game/bot.go
  - testdata/game/golden/seed-42.json
  - internal/play/play.go
  - internal/game/kan.go
  - cmd/play.go
  - internal/game/wall.go
  - internal/game/state.go
  - internal/play/kan_keys.go
  - internal/game/payout.go
tests:
  - internal/game/payout_test.go
  - internal/play/play_test.go
  - internal/game/turn_test.go
  - internal/game/wall_test.go
  - internal/game/bot_test.go
  - internal/game/kan_test.go
  - internal/game/match_test.go
  - internal/play/kan_keys_test.go
  - internal/game/furiten_test.go
-->