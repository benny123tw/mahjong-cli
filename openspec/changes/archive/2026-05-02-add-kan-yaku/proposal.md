## Why

Kan support landed in `add-kan-support`, but the yaku detection table in `internal/riichi/yaku/yaku.go` was written before kan was wired and never grew detectors that depend on `MeldKan` entries. Today a player can declare three or four kans without earning Sankantsu (2 han) or Suukantsu (yakuman); a player can also assemble four concealed triplets — including ankan — without Suuankou (yakuman) firing. The hand-calculator spec even acknowledges Suuankou as "deferred yakuman, not detected in this change". Three new detectors close the gap and make kan support pay off at score time.

- A new value type `hand.CalledMeld{Kind hand.CalledKind, BaseID uint8}` SHALL be added to `internal/riichi/hand/hand.go` along with `CalledKind` enum values `CalledPon`, `CalledChi`, `CalledMinkan`, `CalledAnkan`, `CalledShouminkan`. A new field `CalledMelds []CalledMeld` SHALL be added to `hand.Hand` so the yaku detector can read called-meld metadata that is otherwise lost when `Game.effectiveConcealed` flattens melds into a 14-tile bag. The field's zero value is the empty slice (back-compat: existing fixtures with `CalledMelds = nil` behave exactly as before).
- The engine SHALL populate `hand.Hand.CalledMelds` at the three win-construction sites in `internal/game/turn.go` (the tsumo path, the chankan-ron path, and the regular ron path) by translating each `game.Meld` in `g.melds[s]` into a `hand.CalledMeld` with the appropriate `CalledKind` (mapping `MeldPon → CalledPon`, `MeldChi → CalledChi`, `MeldKan{KanAnkan} → CalledAnkan`, `MeldKan{KanMinkan} → CalledMinkan`, `MeldKan{KanShouminkan} → CalledShouminkan`).
- A new `detectSankantsu` SHALL match when `hand.Hand.CalledMelds` contains exactly three entries with `Kind ∈ {CalledAnkan, CalledMinkan, CalledShouminkan}`, reporting 2 han for both concealed and open hands.
- A new `detectSuukantsu` SHALL match when `hand.Hand.CalledMelds` contains exactly four kan entries (any of the three kan kinds), reporting yakuman; Sankantsu does NOT also match — Suukantsu supersedes (enforced inside `detectSankantsu` by early-returning when the kan count is 4).
- A new `detectSuuankou` SHALL match when the winning decomposition contains four triplet-shaped sets that are all "concealed at agari", where a triplet at base ID `B` is concealed iff (a) no `CalledMeld` with `BaseID == B` exists with `Kind ∈ {CalledPon, CalledMinkan, CalledShouminkan}` (ankan does NOT disqualify), AND (b) the win is by tsumo OR the winning tile did not complete that triplet via shanpon-ron. Reports yakuman; treated as standard (1x) yakuman in v1.
- `Evaluate` in `internal/riichi/yaku/yaku.go` SHALL drop a `Sanankou` match when a `Suuankou` match is also present for the same decomposition (suuankou supersedes sanankou — same pattern already used for ryanpeikou-supersedes-iipeikou).
- The three new detectors SHALL be registered in the `Detectors()` slice in `internal/riichi/yaku/yaku.go` so `calc.Analyze` picks them up automatically.
- The `hand-calculator` capability spec SHALL be updated to add the three rows to the V1 yaku table and add scenarios that document each detector's match condition.

## Non-Goals

- The remaining Group B yakuman (Daisangen, Daisuushii, Shousuushii, Tsuuiisou, Chinroutou, Ryuuiisou, Chuuren poutou) — those are non-kan-related yakuman and SHALL ship in a separate change.
- Multi-yakuman scoring stack interactions (e.g., suuankou + daisangen in one hand). The existing `Award.Total` math handles single-yakuman correctly; multi-yakuman aggregation is a future scoring change.
- Suuankou tanki as a separate **double**-yakuman detector — v1 reports suuankou as a single yakuman regardless of wait shape, matching how other yakuman are already scored. A tanki-vs-shanpon split SHALL be a future scoring tweak.
- Changes to kan declaration or chankan flow — those are already complete in `kan-flow`.
- New `yaku.Context` flags. All three detectors derive from `Decomposition` and `Hand` (now including `Hand.CalledMelds`) alone.
- A general fix to `detectSanankou` for hands containing one minkan plus three concealed triplets — the existing detector returns nil whenever `h.Open` is true, which is a pre-existing limitation. Suuankou treats ankan-aware concealment correctly; the broader sanankou-with-open-melds case SHALL be a separate change.
- Changing `Game.IsHandOpen` to ignore ankans for menzen-tsumo concealment purposes — that touches yaku-orthogonal logic and SHALL ship in its own change if needed.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `hand-calculator`: `Yaku Detection — V1 Set` updated to add Sankantsu, Suukantsu, and Suuankou rows plus their match scenarios.

## Impact

- Affected specs: `hand-calculator` (modified — V1 Set table + scenarios)
- Affected code:
  - Modified: `internal/riichi/hand/hand.go` (new `CalledMeld`/`CalledKind` types + `Hand.CalledMelds` field), `internal/riichi/yaku/yaku.go` (three new detector funcs + slice registration + `Evaluate` supersession rule for suuankou-over-sanankou), `internal/riichi/yaku/yaku_test.go` (per-detector unit tests + suuankou/sanankou interaction tests), `internal/game/turn.go` (populate `Hand.CalledMelds` at the three win-construction sites)
  - New: none — all changes are extensions of existing files
  - Removed: none
