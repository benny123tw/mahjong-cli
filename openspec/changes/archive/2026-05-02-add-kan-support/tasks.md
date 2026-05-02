## 1. Wall: Rinshan Slots (decision: wall rinshan slots are reserved in the dead wall)

- [x] 1.1 Update Wall Rinshan Replacement Draw — add `Wall.RinshanDraw() (tile.Tile, bool)` method to `internal/game/wall.go`. Track a private `kansDrawn int` counter on Wall. The rinshan slots are at dead-wall positions `tiles[len(tiles)-1-2]`, `tiles[len(tiles)-1-3]`, `tiles[len(tiles)-1-4]`, `tiles[len(tiles)-1-5]` (i.e., right after the initial dora indicator at `tiles[len-1]`, leaving the kan-dora indicators at `tiles[len-2-kansDrawn*2]`-style spots — but for v1 simplicity, use `tiles[len(tiles)-2-2*kansDrawn]` as kan-dora and `tiles[len(tiles)-3-2*kansDrawn]` as rinshan). Return `false` once `kansDrawn >= 4`.
- [x] 1.2 Add `Wall.RevealKanDora() tile.Tile` returning the next kan-dora indicator tile (called by `Game.afterKan`). The function reads from the dead wall slot adjacent to the rinshan draw and returns the tile. Use a separate private counter `kanDoraRevealed int` tracking how many kan-dora have been revealed (mirrors `kansDrawn`).
- [x] 1.3 In `internal/game/wall_test.go`, add `TestRinshanDoesNotConsumeLiveWall` constructing a fresh wall, calling `RinshanDraw` once, and asserting `LiveRemaining()` is unchanged at 70.
- [x] 1.4 In `internal/game/wall_test.go`, add `TestRinshanExhaustsAfterFourKans` calling `RinshanDraw` four times successfully then asserting the fifth call returns `(zero, false)`.
- [x] 1.5 In `internal/game/wall_test.go`, add `TestRevealKanDoraReturnsAdjacentDeadWallTile` confirming the reveal slot is distinct from the rinshan slot for the same kan index.

## 2. Engine: Kan State And Inputs (decision: three kan flavors live in `internal/game/kan.go` behind a single entry point)

- [x] 2.1 In `internal/game/state.go`, add `KanKind uint8` enum on `Meld` with values `KanAnkan`, `KanMinkan`, `KanShouminkan` (plus a default zero value for non-kan melds). Add `Meld.KanKind KanKind` field; non-kan melds leave it at zero.
- [x] 2.2 In `internal/game/state.go`, add new input variants: `InputDeclareAnkan{TileID uint8}` and `InputDeclareShouminkan{TileID uint8}`, both implementing the `Input` interface. Add `StateAwaitingChankan{UpgradeTile tile.Tile, Declarer Seat}` implementing the `State` interface.
- [x] 2.3 In `internal/game/turn.go`, add a per-seat `lastDrawWasRinshan [numSeats]bool` field on `Game`. Cleared by every `stepFromAwaitingDiscard` (after a discard or non-tsumo path) and by every successful pon/chi call.
- [x] 2.4 Add `ErrIllegalKan = errors.New("game: illegal kan declaration")` to `internal/game/turn.go`.

## 3. Engine: Kan Handlers in kan.go

- [x] 3.1 Update Ankan Declaration — create `internal/game/kan.go`. Implement `Game.declareAnkan(seat Seat, tileID uint8) error`: count the seat's concealed tiles matching `tileID`; reject with `ErrIllegalKan` if not exactly 4. Remove the four tiles, append a `Meld{Kind: MeldKan, KanKind: KanAnkan, Tiles: [...]}` to `g.melds[seat]`. Call `g.afterKan(seat)`.
- [x] 3.2 Update Minkan Declaration — implement `Game.declareMinkan(claimant Seat, discard tile.Tile, discarder Seat) error`: count claimant's concealed tiles matching `discard.ID`; reject with `ErrIllegalKan` if not exactly 3. Remove the 3 tiles, append `Meld{Kind: MeldKan, KanKind: KanMinkan, Tiles: [discard, ...3 from hand], From: discarder}`. Pop discarder's last pond entry. Set `g.callsHappened = true`. Close all ippatsu windows. Call `g.afterKan(claimant)`.
- [x] 3.3 Update Shouminkan Declaration And Chankan Window (declaration entry) — implement `Game.declareShouminkan(seat Seat, tileID uint8) error`: scan the seat's open melds for an existing `Meld{Kind: MeldPon}` matching `tileID`; reject with `ErrIllegalKan` if absent. Verify the seat's concealed hand contains a tile with the matching ID; reject otherwise. Set state to `StateAwaitingChankan{UpgradeTile: tile.Tile{ID: tileID}, Declarer: seat}`. The actual upgrade and rinshan happen in the chankan resolution.
- [x] 3.4 Update Kan Dora Indicator Reveal — add `Game.afterKan(seat Seat)` shared post-kan helper: call `g.wall.RinshanDraw()`, on `ok=false` return `ErrIllegalKan` (5th kan attempted — the kan is unwound by the caller). On success, append the rinshan tile to `g.hands[seat]`, set `g.lastDrawWasRinshan[seat] = true`, append `g.wall.RevealKanDora()` to `g.doraIndicators` (this is the kan-dora indicator reveal — every kan reveals exactly one new dora indicator), log the kan event, transition to `StateAwaitingDiscard{Player: seat}`.

## 4. Engine: stepFromAwaitingDiscard Wiring And Chankan Resolution

- [x] 4.1 In `internal/game/turn.go`'s `stepFromAwaitingDiscard`, add input cases for `InputDeclareAnkan{TileID}` (call `g.declareAnkan(s.Player, v.TileID)`) and `InputDeclareShouminkan{TileID}` (call `g.declareShouminkan(s.Player, v.TileID)`). Both must check `g.riichiDeclared[s.Player]` first and return `ErrIllegalKan` if true (riichi-and-kan deferred per design).
- [x] 4.2 In `stepFromAwaitingDiscard`, when `s.Player` discards (the existing `InputDiscard` path) AND `g.lastDrawWasRinshan[s.Player]` is true, clear the flag (the rinshan-draw window has closed without a tsumo).
- [x] 4.3 Add a new `Game.stepFromAwaitingChankan(s StateAwaitingChankan, in Input) (Event, error)` method. Accept only `InputResolveClaims`. If a non-declarer submits `Claim{Kind: ClaimRon}` and the ron is valid (calc.Analyze on concealed+UpgradeTile non-nil, not furiten), set state to `StateRoundOver{Outcome: OutcomeRon{Winner, Loser: s.Declarer, Tile: s.UpgradeTile, ...}}` with the calc.Context.Chankan flag passed via `g.contextForWin` (see task 5.x). On no ron (or all rons rejected), upgrade the existing pon meld in place to `KanShouminkan` (find the meld, set `KanKind = KanShouminkan`, append the 4th tile from hand, remove that tile from hand), then call `g.afterKan(s.Declarer)`.
- [x] 4.4 In `internal/game/turn.go`'s `Step` switch, add `case StateAwaitingChankan: return g.stepFromAwaitingChankan(s, in)`.

## 5. Engine: Rinshan And Chankan Context Flag Population (decision: `rinshan` and `chankan` flags populated by engine, not caller)

- [x] 5.1 Update Rinshan And Chankan Context Flags AND Update Group C Game Context Flags — in `Game.contextForWin(winner, isTsumo)`: when `isTsumo` is true and `g.lastDrawWasRinshan[winner]` is true, set `ctx.Rinshan = true` (the engine populates the flag, not the caller). When `isTsumo` is false and the current state is `StateAwaitingChankan`, set `ctx.Chankan = true`. (The chankan check looks at `g.state` since `contextForWin` is called from `stepFromAwaitingChankan` before the state transitions to RoundOver.) This flips the previously-always-false placeholders documented in Group C Game Context Flags. Bot Decision Strategy stays unchanged for kan generation but now correctly observes `Chankan` on shouminkan ron paths.
- [x] 5.2 In `internal/game/turn.go`'s `stepFromAwaitingDiscard` for `InputDeclareTsumo`: after the tsumo succeeds and `lastDrawWasRinshan[s.Player]` was true, clear the flag (it's now consumed by the win).

## 6. Engine: Minkan Path In stepFromAwaitingClaims (decision: shouminkan opens a chankan window as a new state)

- [x] 6.1 In `internal/game/turn.go`'s `stepFromAwaitingClaims`, add a `case ClaimKan:` branch in the existing claim-resolution switch. Call `g.declareMinkan(winner, s.Discard, s.Discarder)`. On `ErrIllegalKan`, return the error. On success, the state was already advanced by `afterKan` to `StateAwaitingDiscard{Player: winner}`.
- [x] 6.2 In `internal/game/call.go`, add `CanKan(hand []tile.Tile, discarded tile.Tile) bool` returning true when the hand contains exactly 3 tiles with the discarded tile's ID. Used by both the engine's claim resolver and the TUI's call-window prompt.
- [x] 6.3 Update the existing `ResolveClaims` in `internal/game/call.go` (or wherever the priority order lives) to honor `ClaimKan` between pon and chi: kan beats pon, pon beats chi. Verify with the existing call-priority tests.

## 7. Engine: Tests for Kan Flow

- [x] 7.1 Create `internal/game/kan_test.go::TestAnkanSucceedsWithFourMatchingTiles`: plant a hand with four `5p`, set state `AwaitingDiscard{Player: <seat>}`, submit `InputDeclareAnkan{TileID: tile.P5}`. Assert the four `5p` are removed, a `Meld{Kind: MeldKan, KanKind: KanAnkan}` is appended, the seat has 14 - 4 + 1 = 11 tiles after the rinshan draw, dora indicator count incremented, state is `AwaitingDiscard{Player: <same seat>}`.
- [x] 7.2 Add `TestAnkanRejectedWithThreeMatchingTiles`: plant a hand with three `5p`, submit ankan — assert returns `ErrIllegalKan` and state is unchanged.
- [x] 7.3 Add `TestMinkanWinsOverPon`: plant two seats — one with three `5p` (kan-eligible), another with two `5p` (pon-eligible). Drive `StateAwaitingClaims{Discard: 5p, Discarder: SeatEast}` with both claims. Assert the kan-claimant's hand has the new `KanMinkan` meld and the pon-claimant's hand is unchanged.
- [x] 7.4 Add `TestShouminkanCompletesWhenNoChankan`: plant a seat with an open `MeldPon` for `5p` and a `5p` in hand; set state `AwaitingDiscard{Player: <seat>}`. Submit `InputDeclareShouminkan{TileID: tile.P5}`. Assert state transitions to `AwaitingChankan{...}`. Submit `InputResolveClaims{Claims: nil}`. Assert the pon is upgraded to `KanShouminkan` (4-tile meld), rinshan is drawn, and state returns to `AwaitingDiscard{Player: <declarer>}`.
- [x] 7.5 Add `TestChankanRonPreemptsShouminkan`: plant a tenpai opposing seat that wins on `5p`; drive shouminkan on `5p`; submit `Claim{Kind: ClaimRon}` from the opposing seat. Assert the round terminates as `OutcomeRon{Winner, Loser: <declarer>, Tile: 5p}` AND the winning context's `Chankan` flag was true (assert via inspecting the calc.Result's yaku list — chankan should appear).
- [x] 7.6 Add `TestRinshanTsumoSetsRinshanFlag`: plant a hand that will win on its 14th tile, declare ankan on that hand (use a hand where 4 of one tile + a 9-tile partial winning shape misses by 1, and the rinshan-replacement-tile completes the hand). Submit `InputDeclareTsumo`. Assert the calc.Result's yaku list contains "rinshan kaihou".
- [x] 7.7 Add `TestKanRejectedWhenInRiichi`: plant a riichi-declared seat with four `5p` (manually set `g.riichiDeclared[seat] = true` via test helper). Submit ankan — assert `ErrIllegalKan` and state is unchanged.
- [x] 7.8 Add `internal/game/wall_test.go::TestRinshanDoesNotConsumeLiveWall` (covered by task 1.3 — verify this is the same test).

## 8. TUI: Kan Key Wiring (decision: riichi-and-kan interaction: greyed out for now)

- [x] 8.1 Update Keybinding Map — in `internal/play/play.go`'s `handleKey`, replace the existing `case "k"` placeholder. When state is `StateAwaitingDiscard{Player: HumanSeat}`: scan the human's hand for any 4-of-a-kind concealed (eligible for ankan) AND scan their open melds for any `MeldPon` whose tile ID matches a tile in their concealed hand (eligible for shouminkan). If `Game.IsRiichiDeclared(HumanSeat)` is true, set ackText to "kan: not allowed during riichi" and return. If at least one option exists, set a `Model.kanPickerOpen bool` flag (or store the option list) and return — the rendered footer shows the picker.
- [x] 8.2 Implement the picker selector: when `kanPickerOpen` is true and the human presses `1`–`9`, map to the nth kan option and submit `InputDeclareAnkan{TileID}` or `InputDeclareShouminkan{TileID}` to the engine. Clear `kanPickerOpen` after submission. On engine error, set ackText accordingly.
- [x] 8.3 In `handleKey` for `StateAwaitingClaims`: when the human presses `k` and `game.CanKan(humanHand, cs.Discard)` returns true, submit `InputResolveClaims{Claims: map[Seat]Claim{HumanSeat: {Kind: ClaimKan}}}`. On error, set ackText.
- [x] 8.4 Render the picker in the footer: when `kanPickerOpen`, replace the normal action footer with `[1] 5p ankan  [2] 7m shouminkan  [Esc] cancel` (numbered list of options). Esc cancels the picker.
- [x] 8.5 Update `RenderCallFooter` to show `[K]on` live when `game.CanKan(humanHand, cs.Discard)` returns true; otherwise greyed.

## 9. TUI: Tests

- [x] 9.1 Create `internal/play/kan_keys_test.go::TestKanKeyOpensPickerWhenAnkanAvailable`: plant a human hand with four `5p`, set state `AwaitingDiscard{Player: HumanSeat}`. Send `tea.KeyPressMsg{Code: 'k'}`. Assert `kanPickerOpen` is true and the rendered View contains "5p" and a numbered selector.
- [x] 9.2 Add `TestKanPickerSubmitsAnkan`: starting from the picker-open state, send `tea.KeyPressMsg{Code: '1'}`. Assert the engine state shows the ankan completed (4 fewer `5p`, new MeldKan, state still `AwaitingDiscard{Player: HumanSeat}`).
- [x] 9.3 Add `TestKanKeyGreyedDuringRiichi`: plant the human in riichi (`SetTestRiichiDeclared(HumanSeat, true)`) with four `5p`. Send `tea.KeyPressMsg{Code: 'k'}`. Assert `kanPickerOpen` remains false and ackText contains "riichi".
- [x] 9.4 Add `TestKanKeySubmitsMinkanInClaimWindow`: plant the human with three `5p`, set state `AwaitingClaims{Discard: 5p, Discarder: SeatEast}`. Send `tea.KeyPressMsg{Code: 'k'}`. Assert state advances to `AwaitingDiscard{Player: HumanSeat}` (the kan succeeded).
- [x] 9.5 Add `TestBotRonsOnHumanShouminkanChankan`: plant the human with an open MeldPon for `5p` and a `5p` in hand, plant a bot tenpai on `5p`, drive the shouminkan declaration, drive the bot tick. Assert the round terminates as `OutcomeRon` with the bot as winner and the calc.Result's yaku list contains "chankan".

## 10. Verification

- [x] 10.1 Confirm `TestAnkan*`, `TestMinkan*`, `TestShouminkan*`, `TestChankan*`, `TestRinshan*`, `TestKan*` exist across `internal/game/kan_test.go`, `internal/game/wall_test.go`, and `internal/play/kan_keys_test.go`.
- [x] 10.2 Run `go test ./...` and confirm all suites pass (including the existing single-hand and match-flow tests).
- [x] 10.3 Run `golangci-lint run ./...` and confirm 0 issues.
- [x] 10.4 Run `spectra validate add-kan-support` and confirm valid.
