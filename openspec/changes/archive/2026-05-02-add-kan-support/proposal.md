## Why

Kan has been a deferred placeholder since the game-loop spec shipped: the `K` key shows "kan: not supported in v1 (deferred to add-kan-support)", `MeldKan` is in the meld kind enum but never produced, and `Rinshan`/`Chankan` flags in `calc.Context` are wired through but always false. To complete the playable rule set, the human needs to be able to declare a concealed kan (ankan) on their own turn, the engine needs to draw a replacement tile from the dead wall and reveal an additional dora indicator, and the yaku detector needs to actually populate `Rinshan` so the rinshan kaihou yaku trigger from a kan-replacement-tile-draw fires. Open kans (minkan from an opponent discard, and shouminkan upgrading an open pon plus the chankan ron window) round out the rule set so the human can actually use kan offensively.

## What Changes

- New input variants: `InputDeclareAnkan{TileID uint8}` (own turn, concealed 4-of-a-kind) and `InputDeclareShouminkan{TileID uint8}` (own turn, the player has an open pon and now holds the 4th matching tile).
- New claim kind: `ClaimKan` (already in the enum, never honored). The engine's `ResolveClaims` priority already lists it between pon and chi; this change wires the actual minkan flow.
- Engine state machine: `StateAwaitingDiscard` accepts `InputDeclareAnkan` and `InputDeclareShouminkan` inputs in addition to the existing discard / tsumo / riichi paths. `StateAwaitingClaims` accepts `ClaimKan` for minkan calls. After any successful kan, the engine: (a) replaces the consumed tiles with a `MeldKan` meld of the appropriate `KanKind`, (b) draws a replacement tile from the dead wall (the `rinshan` slot), (c) reveals the next dora indicator, (d) for shouminkan, opens a chankan window so other seats can ron on the upgraded tile.
- Replacement-draw plumbing: `Wall.RinshanDraw() (tile.Tile, bool)` reserves four dead-wall tiles for kan replacements (alongside the existing dora indicator). `LiveRemaining()` semantics unchanged (the rinshan tiles were already inside the 14-tile dead-wall reservation).
- Per-seat rinshan-tile flag: when a player's most recent draw was via a kan replacement, `Game.contextForWin` populates `calc.Context.Rinshan = true` for any subsequent tsumo declaration on that draw. Cleared on the next discard or call.
- Chankan flag: when a seat declares shouminkan, the engine opens an `AwaitingClaims`-shaped window with the would-be-added tile as the discard. If a non-declaring seat submits `ClaimRon`, `calc.Context.Chankan = true` is set and the kan does NOT go through (the ron pre-empts). Otherwise the kan completes.
- Dora-indicator reveals: each successful kan reveals an additional indicator from the dead wall. `Game.DoraIndicators()` returns the cumulative list.
- Bot policy unchanged: bots still never declare kan (Bot.ShouldKan returns false); minkan calls from bots are not generated. Bots CAN ron on a human's chankan (the existing dispatchBotClaims path inspects the chankan window through the same mechanism).
- TUI: the `K` key in `StateAwaitingDiscard{Player: HumanSeat}` opens a one-shot ankan/shouminkan picker — when the human's hand contains either a concealed 4-of-a-kind OR an open pon plus the 4th tile, the picker shows the eligible tile IDs and any keypress 1..N selects one. The footer ack text reports successful kan declarations and rinshan replacement draws.

## Capabilities

### New Capabilities

- `kan-flow`: kan-call orchestration — concealed/open/added-kan declarations, dead-wall rinshan replacement draw, kan-dora indicator reveal, chankan claim window for shouminkan upgrades.

### Modified Capabilities

- `game-loop`: `Bot Decision Strategy` updates (kan: still never; chankan ron eligibility); `Group C Game Context Flags` updates (`Rinshan` and `Chankan` are now actually populated, replacing the "always false in v1" placeholders); `Round Termination and Outcome` adds the chankan ron path on shouminkan upgrades.
- `play-screen`: `Keybinding Map` updates so `K` is no longer a hardcoded "not supported" placeholder; new entries for the ankan/shouminkan picker.

## Impact

- New: `internal/game/kan.go` (kan declaration handlers, rinshan draw helper, kan-dora reveal logic), `internal/game/kan_test.go`, `internal/play/kan_keys.go` (the K-key picker UI), `internal/play/kan_keys_test.go`.
- Modified: `internal/game/wall.go` (RinshanDraw helper + dead-wall slot accounting), `internal/game/state.go` (new Input variants `InputDeclareAnkan`, `InputDeclareShouminkan`, new `KanKind` enum on Meld), `internal/game/turn.go` (extended discard-state input handling, contextForWin populates Rinshan/Chankan, post-kan dora reveal), `internal/game/call.go` (CanKan check for minkan eligibility, kan-from-claim handler), `internal/play/play.go` (K key wiring + chankan claim integration), `internal/riichi/score/fu.go` (verify ankan/minkan/shouminkan fu values still match — likely no change, but audited).
- Removed: the `case "k"` placeholder ack text in `internal/play/play.go`.
