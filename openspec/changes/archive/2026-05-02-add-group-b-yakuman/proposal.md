## Why

The kan-aware yakuman tier landed in `add-kan-yaku`, but seven shape/composition yakuman were explicitly deferred: Daisangen, Daisuushii, Shousuushii, Tsuuiisou, Chinroutou, Ryuuiisou, and Chuuren poutou. None of them depend on kan kind — they are all checks over `Decomposition`, the concealed tile bag, or the called-meld set already exposed via `hand.Hand`. With those seven shipped, the v1 yakuman roster is complete (kokushi + tenhou + chiihou + suuankou + suukantsu plus these seven) and `mahjong calc` will correctly score every documented yakuman shape.

## What Changes

- A new `detectDaisangen` SHALL match when the winning decomposition contains triplet-shaped sets (`MeldTriplet` in `d.Sets()`) at all three dragon bases: `tile.Haku`, `tile.Hatsu`, and `tile.Chun`. Reports yakuman. Open-ok.
- A new `detectDaisuushii` SHALL match when the winning decomposition contains triplet-shaped sets at all four wind bases: `tile.EastWind`, `tile.SouthWind`, `tile.WestWind`, `tile.NorthWind`. Reports yakuman. Open-ok. Some rulesets score this as double yakuman; v1 reports it as standard (1x) yakuman to match the rest of the table.
- A new `detectShousuushii` SHALL match when the winning decomposition contains triplet-shaped sets at exactly three wind bases AND the pair is the fourth wind. Reports yakuman. Open-ok. `Evaluate` SHALL drop a `Shousuushii` match when `Daisuushii` is also present (cross-cutting supersession rule, mirroring the existing Suuankou-supersedes-Sanankou and Ryanpeikou-supersedes-Iipeikou blocks).
- A new `detectTsuuiisou` SHALL match when every tile in `h.Concealed` is an honor (`tile.Tile.IsHonor()` returns true). Works for both `FormStandard` and `FormChiitoitsu`. Reports yakuman. Open-ok.
- A new `detectChinroutou` SHALL match when every tile in `h.Concealed` is a numeric terminal (rank 1 or 9, not an honor) AND the form is `FormStandard`. Chiitoitsu is impossible because there are only six terminal tile IDs (1m, 9m, 1p, 9p, 1s, 9s) and chiitoitsu needs seven distinct pairs. Reports yakuman. Open-ok.
- A new `detectRyuuiisou` SHALL match when every tile in `h.Concealed` has an ID drawn from the green-tile set `{tile.S2, tile.S3, tile.S4, tile.S6, tile.S8, tile.Hatsu}`. Works for `FormStandard` (chii sequences are constrained — only 2s3s4s qualifies). Reports yakuman. Open-ok.
- A new `detectChuurenPoutou` SHALL match when (a) the hand is concealed (`!h.Open` AND `len(h.CalledMelds) == 0`), (b) every tile in `h.Concealed` belongs to a single numeric suit, AND (c) the per-rank counts within that suit form the pattern `[3, 1, 1, 1, 1, 1, 1, 1, 3]` PLUS one extra tile of any rank from 1..9 in that suit (the winning tile bringing the 14-tile bag to a 1-1-1-2-3-4-5-6-7-8-9-9-9 + N shape). Reports yakuman. Concealed-only — does not match if any `CalledMeld` is present.
- All seven detectors SHALL be registered in the `Detectors()` slice in `internal/riichi/yaku/yaku.go`.
- The `hand-calculator` capability spec SHALL be updated to add the seven rows to the V1 yaku table and per-detector scenarios documenting each match condition.

## Non-Goals

- Multi-yakuman scoring stack interactions (e.g., daisangen + tsuuiisou in one hand). The existing `Award.Total` math handles single-yakuman correctly; multi-yakuman aggregation is a separate scoring change.
- Double-yakuman variants — Daisuushii (some rulesets), Suuankou-tanki, Kokushi-thirteen-wait. All ship as standard (1x) yakuman in v1 to match the rest of the existing yakuman table.
- Local-rule yakuman like Renhou (human win on a non-dealer's first uninterrupted turn after no calls). Renhou requires turn-state tracking similar to Tenhou/Chiihou and SHALL ship in its own change if needed.
- Sanrenkou (three consecutive triplets), Iipinmoyue, Chuuren-poutou-pure-9-wait, and other local-rule non-yakuman additions.
- Any changes to kan support, calc CLI flags, or the play-screen TUI — the seven detectors plug into existing infrastructure.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `hand-calculator`: `Yaku Detection — V1 Set` updated to add Daisangen, Daisuushii, Shousuushii, Tsuuiisou, Chinroutou, Ryuuiisou, and Chuuren poutou rows plus their match scenarios.

## Impact

- Affected specs: `hand-calculator` (modified — V1 Set table + scenarios)
- Affected code:
  - Modified: `internal/riichi/yaku/yaku.go` (seven new detector funcs + `Detectors()` registration + Daisuushii-supersedes-Shousuushii rule in `Evaluate`), `internal/riichi/yaku/yaku_test.go` (per-detector unit tests)
  - New: none — all changes are extensions of existing files
  - Removed: none
