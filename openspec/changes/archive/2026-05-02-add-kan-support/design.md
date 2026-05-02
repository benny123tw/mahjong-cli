## Context

Today's engine handles the three most common riichi calls — pon, chi, ron — but kan has been a placeholder since the v1 design. The state machine has the building blocks: `MeldKan` is enumerated (alongside `MeldPon`/`MeldChi`), `ClaimKan` is in the claim-kind enum, and `Rinshan`/`Chankan` flags exist in `calc.Context`. The yaku detector already scores rinshan kaihou and chankan when the flags are true. None of these pieces are connected: bots never declare kan, the human's `K` key shows a "not supported" message, the call-window resolver doesn't honor `ClaimKan`, and the wall doesn't have a rinshan-draw helper.

There are three kan flavors in riichi:

- **Ankan (concealed kan)** — declared during own discard turn; player has 4 of a tile in hand. Stays "concealed" for yaku/fu purposes (concealed-hand yaku like menzen tsumo and pinfu remain possible if the rest of the hand is also concealed; pinfu specifically requires all sequences and is incompatible with kan, but other concealed-only yaku survive). Triggers rinshan replacement draw + extra dora.
- **Minkan / Daiminkan (open kan from opponent discard)** — claimed in the post-discard claim window when the player has 3 of the discarded tile. Like a pon but consumes 4 tiles total. Opens the hand. Triggers rinshan replacement draw + extra dora.
- **Shouminkan / Added kan (upgrade open pon)** — own discard turn; player has an open `MeldPon` and now holds the 4th matching tile. Upgrades the meld in place. Opens a chankan claim window: any other seat that would win on the upgraded tile can ron and pre-empt the kan. If no chankan ron, the kan completes with rinshan replacement draw + extra dora.

The dead wall (last 14 tiles of the 136-tile wall) reserves space for up to 4 kan replacement tiles plus dora indicators. The rinshan slot is fixed at the dead wall — drawing rinshan does NOT consume from the live wall, so `LiveRemaining()` is unaffected. After a kan, an additional dora indicator is revealed (which includes the original indicator's neighbour in the dead-wall layout).

## Goals / Non-Goals

**Goals:**

- Human can declare ankan on their own turn via the `K` key.
- Human can declare shouminkan (upgrade an open pon) via the same `K` key when eligible.
- Human can declare minkan in the call window when an opponent discards a tile they hold 3 of, via the `K` key.
- Engine performs the replacement draw from the dead wall and reveals an additional dora indicator on every successful kan.
- `calc.Context.Rinshan` is set to true for tsumo declarations on a kan-replacement draw.
- `calc.Context.Chankan` is set to true for ron declarations on a shouminkan upgrade tile.
- Engine state machine accepts kan inputs/claims and remains deterministic across replays.

**Non-Goals:**

- Bot kan declarations (bots continue to never declare any flavor of kan; they CAN ron via chankan because that's a normal ron path).
- Sankantsu / suukantsu yakuman detection (deferred — the dora indicator and rinshan plumbing are wired but yakuman scoring beyond the existing kokushi is out of scope).
- Multiple kan-dora reveals timing variations (we always reveal immediately after the kan, not at next discard — minor rule variation).
- Kan-with-riichi-already-declared restrictions (real riichi: a riichi player can only ankan if it doesn't change the hand's wait; we don't enforce this — riichi player simply cannot kan in v1, the K key is greyed when riichi is active).
- Shouminkan upgrades that change the wait shape (similar restriction; deferred — the player can shouminkan any matching open pon).

## Decisions

### Three Kan Flavors Live In `internal/game/kan.go` Behind A Single Entry Point

A new file `internal/game/kan.go` houses the three kan declaration handlers:

- `Game.declareAnkan(seat Seat, tileID uint8) error` — validates 4-of-a-kind in hand, removes the four tiles, builds a `Meld{Kind: MeldKan, KanKind: KanAnkan, Tiles: [4]tile.Tile{...}}`, calls the shared `Game.afterKan(seat)` helper for replacement draw + dora.
- `Game.declareMinkan(claimant Seat, discard tile.Tile, discarder Seat) error` — validates 3-of-a-kind in claimant's hand, consumes the 3 + adds the discard, builds a `Meld{Kind: MeldKan, KanKind: KanMinkan, From: discarder}`, calls `afterKan`.
- `Game.declareShouminkan(seat Seat, tileID uint8) error` — validates the seat has an existing open `MeldPon` for `tileID` AND has the 4th matching tile in hand. Upgrades the meld in place (changes `KanKind` from "n/a" to `KanShouminkan`, appends the 4th tile). Opens a chankan claim window via state transition to a new `StateAwaitingChankan{Tile, Declarer}` (similar to `AwaitingClaims` but only `ClaimRon` is honored). On no-ron, calls `afterKan`.

`Game.afterKan(seat Seat)` is the shared post-kan flow: increments the kan counter, pulls a tile via `wall.RinshanDraw()`, reveals one more dora indicator, sets the seat's `lastDrawWasRinshan = true` flag (cleared on next discard or call), and transitions to `StateAwaitingDiscard{Player: seat}`. The seat now holds 14 tiles (13 hand + rinshan replacement) and must discard or tsumo.

`KanKind` is a new enum on `Meld`: `KanAnkan`, `KanMinkan`, `KanShouminkan`. The existing `MeldKind` (`MeldPon`/`MeldChi`/`MeldKan`) stays — `KanKind` is a sub-discriminator only relevant when `MeldKind == MeldKan`.

**Alternative considered:** Three separate state machines per kan flavor. Rejected — they share post-kan plumbing (rinshan, dora reveal, return to discard state). One shared `afterKan` helper keeps the variants DRY.

### Wall Rinshan Slots Are Reserved In The Dead Wall

The wall already reserves a 14-tile dead wall (`deadWallSize = 14`). Of those, slots `[0..3]` (the four tiles farthest from the live-wall boundary) are the rinshan replacement tiles, indexed by kan count. The dora indicator currently lives at `tiles[len-1]`; subsequent kan-dora indicators live at `tiles[len-2]`, `tiles[len-3]`, etc.

`Wall.RinshanDraw() (tile.Tile, bool)` returns the next rinshan tile by kan count; returns `false` after 4 kan (rule: max 4 kan per round). `Wall.RevealKanDora()` increments a kan-dora counter and returns the newly-revealed indicator (which is appended to `Game.doraIndicators`).

**Alternative considered:** Pull the rinshan from the live wall (decreasing `LiveRemaining()`). Rejected — that's incorrect riichi: the live wall must always have its 70-tile budget; kans pull from the dead-wall reserve.

### Shouminkan Opens A Chankan Window As A New State

Shouminkan is the only call that creates a "third party can interrupt" moment outside the normal post-discard claim window. To handle this cleanly:

- New state `StateAwaitingChankan{UpgradeTile tile.Tile, Declarer Seat}` is entered after a successful shouminkan declaration, before the kan completes.
- Only `InputResolveClaims` is accepted in this state, but only `ClaimRon` claims are honored (pon/chi/kan are ignored on a chankan window).
- If a non-declarer submits `ClaimRon` and the chankan-ron is valid (yaku-bearing, not furiten), the round ends as `OutcomeRon` with `chankan = true` populated in the winning context. The shouminkan does NOT complete.
- If no valid ron, `Game.afterKan(declarer)` runs and the round continues.

The existing `dispatchBotClaims` is reused for chankan windows: bots evaluate ron normally, and the chankan flag flows in through `Game.contextForWin` because the engine knows the in-flight upgrade tile.

**Alternative considered:** Reuse `StateAwaitingClaims` with a discriminator field. Rejected — the semantics differ (only ron honored, no pon/chi possible since the tile is being added to a pon, not discarded). A distinct state makes the constraint explicit and prevents accidental pon/chi paths.

### `Rinshan` And `Chankan` Flags Populated By Engine, Not Caller

The existing `calc.Context.Rinshan` and `calc.Context.Chankan` flags have lived as "always false in v1" placeholders. With kan support:

- `Rinshan` is populated by `Game.contextForWin(winner, isTsumo=true)` when `g.lastDrawWasRinshan[winner]` is true. Cleared on the next discard or call.
- `Chankan` is populated by `Game.contextForWin(winner, isTsumo=false)` when the winning ron was from a shouminkan upgrade window (i.e., the engine entered `StateAwaitingChankan` and the winner submitted ClaimRon). The discarder field on `OutcomeRon` is the shouminkan declarer.

**Alternative considered:** Per-input `IsRinshan`/`IsChankan` flags on `InputDeclareTsumo`/`ClaimRon`. Rejected — the engine has the authoritative knowledge (it knows whether the seat's last draw was rinshan, it knows whether the current state is a chankan window). Pushing the determination to the caller risks divergence.

### Riichi-And-Kan Interaction: Greyed Out For Now

Real riichi has nuanced rules about a riichi-declared player kan-ing: only ankan is legal, and only if the kan doesn't change the hand's machi shape. We don't enforce this in v1.

The simpler v1 rule: **if the human has declared riichi, the K key is greyed and unavailable**. Kan declarations only work pre-riichi. This is conservative — it forbids legal-but-rare kans rather than allow illegal ones.

**Alternative considered:** Implement the full machi-preservation check. Deferred — requires comparing pre- and post-kan machi sets; substantial logic for an edge case the user (a TW-mahjong player learning JP riichi) is unlikely to encounter early.

## Risks / Trade-offs

[Risk: rinshan replacement and live wall draw counter could drift if `Wall.Draw()` and `Wall.RinshanDraw()` don't coordinate on dead-wall vs live-wall slots] → Mitigation: add `wall_test.go::TestRinshanDoesNotConsumeLiveWall` asserting `LiveRemaining()` is unchanged before and after a `RinshanDraw`.

[Risk: chankan window state change could leak — if the engine enters StateAwaitingChankan and the tea program crashes, the upgraded pon meld is in an inconsistent state] → Mitigation: don't mutate the meld until AFTER the chankan window resolves with no ron. Keep the upgrade tile separate (in `StateAwaitingChankan{UpgradeTile}`) and apply it only on the no-ron path.

[Risk: dora-indicator order across multi-kan in one hand could differ from real riichi's "reveal after kan completes" timing] → Mitigation: we always reveal immediately after the rinshan draw (inclusive of any subsequent shouminkan-after-pon plumbing). This is one of the standard rule variations and matches modern online clients.

[Risk: TUI K-key picker UX — if the human has multiple eligible kan options (e.g., two different 4-of-a-kinds), how do they pick?] → Mitigation: the picker shows numbered options; pressing 1..N selects. With at most ~3 unique 4-of-a-kinds plus shouminkan options on a 14-tile hand, this fits in a single footer line.

[Risk: bot ron on chankan needs an integration test or it'll silently miss] → Mitigation: add `internal/play/kan_keys_test.go::TestBotRonsOnHumanShouminkanChankan`: plant a bot tenpai on the upgrade tile, drive the human's shouminkan, assert the bot rons with `Chankan` populated.
