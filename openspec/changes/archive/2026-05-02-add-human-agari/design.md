## Context

The engine state machine has five states (`AwaitingDraw`, `AwaitingDiscard`, `AwaitingClaims`, `RoundOver`, `GameOver`) with pure transition functions. Group C yaku detection (`Ippatsu`, `DoubleRiichi`, `Tenhou`, `Chiihou`, `Haitei`, `Houtei`, `Rinshan`, `Chankan`) is plumbed end-to-end via `Game.contextForWin → calc.Context → calc.Analyze`. The four turn-aware flags that don't require kan support (`Tenhou`, `Chiihou`, `Haitei`, `Houtei`) are already populated; the riichi-dependent flags (`Ippatsu`, `DoubleRiichi`) stay false because no caller ever sets `Game.riichi`.

The TUI's `RenderCallFooter` hardcodes `[K]an (greyed)` and `[R]on (greyed)` as placeholder labels. The `r` key currently sets `ackText = "riichi: not implemented in v1"`. The human's claim window only handles pon, chi, and pass.

Constraints:
- Engine remains UI-agnostic (no imports from internal/play).
- All state mutations route through `Game.Step`.
- Determinism contract: same seed produces byte-identical event log. Adding riichi state must not perturb non-riichi runs.
- Spec language is normative (SHALL / MUST). Scenarios are GIVEN / WHEN / THEN.

## Goals / Non-Goals

**Goals:**

- Human player can declare riichi and win by ron from the call window.
- Group C `Ippatsu` and `DoubleRiichi` detection paths are exercised end-to-end (riichi state actually flips, ippatsu window tracks correctly).
- Furiten prevents an illegal ron and the call footer surfaces why ron is unavailable.
- Footer labels for ron / riichi are live (no hardcoded `(greyed)` strings); the kan label stays placeholder until kan support lands.

**Non-Goals:**

- Bot riichi declaration and bot ron from claim window (deferred to add-smart-ai).
- Pao / sekinin-barai liability rules.
- Temporary furiten across multiple opponent discards (only permanent furiten — machi tile in own pond — implemented).
- Kan-after-riichi.
- Riichi confirmation prompt (single keystroke commits; player avoids by pressing `d` instead).

## Decisions

### Per-Seat Riichi State Lives On Game As Three Parallel Slices

Three parallel `[numSeats]<T>` arrays on `Game`:
- `riichiDeclared [numSeats]bool` — true once the seat has declared riichi this round.
- `riichiTurn [numSeats]int` — the discard count at the moment of declaration. Used to compute "this seat's next draw" boundary for ippatsu.
- `ippatsuLive [numSeats]bool` — true between declaration and either (a) the next own draw, or (b) any call from any seat. When the seat wins while this is true, `Ippatsu = true`.

**Alternative considered:** A single `riichiSeats map[Seat]riichiState` map. Rejected — the bookkeeping is exactly four slots, fixed cardinality, so a parallel array is simpler and cheaper.

**Alternative considered:** A separate `RiichiState` struct embedded in `Game`. Rejected for the same reason — the three fields are independent and it's clearer to read `g.ippatsuLive[s]` than `g.riichi.ippatsuLive[s]`.

### Combined `InputDiscard{Index, Riichi: bool}` — Not A Separate `InputDeclareRiichi`

Riichi declaration in real riichi is atomic with the discard that puts the seat into tenpai-and-locked-in. Modeling it as two inputs (`InputDeclareRiichi` then `InputDiscard`) creates an awkward intermediate state where the seat has "declared riichi but not yet discarded" and we'd need a sixth state or a flag on `StateAwaitingDiscard`.

Decision: extend `InputDiscard` with an optional `Riichi bool` field. When `Riichi: true`, the engine validates the four legality preconditions (concealed, ≥1000 points, ≥4 wall, post-discard tenpai) before applying the discard. If any precondition fails, return a new sentinel error `ErrIllegalRiichi` and leave state unchanged.

**Alternative considered:** Separate `InputDeclareRiichi{Index int}` that combines declaration and discard internally. Rejected — same effective semantics, but it adds a new Input type for what is fundamentally "discard with a flag set". The Riichi field on InputDiscard is more discoverable.

### Riichi-Restricted Discard Enforced By Comparing Index Against Just-Drawn Tile

After riichi is declared, subsequent discards SHALL only be the just-drawn tile. The just-drawn tile is always at `len(hand)-1` for the human (with the sort invariant) and the engine has no other notion of "which tile was drawn".

Implementation: when handling `InputDiscard` from a seat with `riichiDeclared[s] = true`, validate `v.Index == len(g.hands[s])-1`. If not, return `ErrIllegalDiscard`. The TUI, in turn, restricts the cursor to index 13 when state shows the seat is post-riichi.

**Alternative considered:** Tracking `lastDrawnTile [numSeats]tile.Tile` and validating by tile identity. Rejected — slot index is sufficient, and tile identity introduces the question of "what if a duplicate tile already in hand happens to match the just-drawn tile" which is needless ambiguity.

### Permanent Furiten Implementation Walks The Seat's Own Pond Once Per Query

`IsFuriten(seat)` returns true if any tile in `Game.Discards(seat)` matches a tile ID in the seat's machi (computed by `hand.Machi`). v1 only implements permanent furiten — the temporary kind (passed on an opponent's machi tile since last own draw) requires per-seat "did I see a winning tile pass since my last draw" tracking, which lands when bot ron does (add-smart-ai).

Cost: O(pond-size × machi-size) per query. Pond max 24 tiles (single-round), machi typically 1-3 tiles. Negligible.

**Alternative considered:** Cache a `[numSeats]map[uint8]bool` of "tiles in own pond by ID" and update on every discard. Rejected — premature optimization. The machi computation in `hand.Machi` already dominates.

### Double Riichi Detection reuses existing `noPriorDiscards` and `!callsHappened` logic

`DoubleRiichi` is "riichi declared on the seat's first uninterrupted draw". Existing logic in `contextForWin` already detects "no prior discards anywhere" via `g.noPriorDiscards()` and "no calls happened" via `!g.callsHappened`.

Decision: when `InputDiscard{Riichi: true}` succeeds, store `doubleRiichi[seat] = !callsHappened && noPriorDiscards()`. Surface that in `contextForWin` so the calc context's `DoubleRiichi` flag is set when the seat wins.

When `DoubleRiichi: true`, the calc package already suppresses normal `Riichi` from the yaku list (verified in existing yaku tests). So we set both flags and trust the calc layer to dedupe.

### Furiten Hint Replaces Footer Item Greyed Style

Today the footer renders five items joined by two spaces. When ron is illegal because of furiten, the player needs to know *why* — greying alone is ambiguous (it could mean "no winning shape" or "furiten"). 

Decision: when `IsFuriten(Human) == true` AND `calc.Analyze` would have returned non-nil, render `[R]on` greyed with a trailing `(furiten)` suffix. When `IsFuriten == false` AND `calc.Analyze` returns non-nil, render `[R]on` live. When `calc.Analyze` returns nil, render greyed without suffix (the player knows the hand isn't winning).

**Alternative considered:** A separate ackText line that says "furiten — cannot ron". Rejected — the footer already groups call-window state and adding another line crowds the 24-row budget.

### Tests Expand Existing Files Plus One New File

- `internal/game/turn.go` gains the new logic; tests for it go in `internal/game/riichi_test.go` (new file) — six focused tests (declare legal, declare illegal-because-funds, declare illegal-because-noisy-tenpai, ippatsu window opens, ippatsu window closes on call, double-riichi flag).
- `internal/play/play_test.go` gains four tests (R in claims = ron, R in discard = riichi, footer shows live `[R]on` when canRon, footer shows `(furiten)` suffix when furiten).
- The golden fixture (`testdata/game/golden/seed-42.json`) does not change because the golden test doesn't exercise riichi or human ron — it auto-discards index 0 for every seat with no claims.

## Risks / Trade-offs

[Risk: combined `InputDiscard{Riichi}` makes the Input shape less symmetric — the call resolution `InputResolveClaims` is simple, but `InputDiscard` now has an optional flag] → Mitigation: zero-value default keeps existing callers (`InputDiscard{Index: 0}` etc.) unchanged. The Riichi field is additive and silently false unless explicitly set.

[Risk: riichi-restricted-discard enforcement uses index check, which depends on the sort invariant placing the drawn tile at index 13. If the sort invariant breaks (e.g., a future change re-sorts after draw), the restriction breaks silently] → Mitigation: encode the invariant as an assertion in the post-draw transition — the drawn tile lives at `len(hand)-1`. This already holds because draw appends without sorting (per add-hand-sort's spec).

[Risk: permanent furiten alone is a half-correct implementation; the player might press R thinking they can ron when in fact they're in temporary furiten] → Mitigation: documented as v1 limitation. Temporary furiten lands with bot ron in add-smart-ai. The hint says "(furiten)" only for permanent cases; ambiguous greyed-no-suffix means "no win available".

[Risk: ippatsu window logic depends on observing every call. The current code sets `g.callsHappened = true` only on pon and chi — kan calls aren't observed because kan isn't supported. When kan lands, the kan handler MUST also set `callsHappened` and break ippatsu] → Mitigation: add a comment in the kan-deferred site of `stepFromAwaitingClaims` flagging this dependency.
