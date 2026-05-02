## 1. Daisangen — three dragon triplets

- [x] 1.1 In `internal/riichi/yaku/yaku_test.go`, add a failing test `TestDetectDaisangenAllThreeDragons` that builds a winning hand with triplets at `tile.Haku`, `tile.Hatsu`, `tile.Chun` plus an extra triplet and pair (e.g. `5z5z5z6z6z6z7z7z7z3m3m3m4m4m`), then asserts the result contains `Match{Name: "Daisangen", IsYakuman: true}` per Yaku Detection — V1 Set.
- [x] 1.2 In `internal/riichi/yaku/yaku.go`, add `func detectDaisangen(d hand.Decomposition, h hand.Hand, ctx Context) []Match` that returns a yakuman match when `d.Sets()` contains a `MeldTriplet` at each of the three dragon bases. Register it in `Detectors()`.

## 2. Daisuushii — four wind triplets

- [x] 2.1 In `internal/riichi/yaku/yaku_test.go`, add a failing test `TestDetectDaisuushiiAllFourWinds` that builds a winning hand with triplets at all four wind bases plus a pair (e.g. `1z1z1z2z2z2z3z3z3z4z4z4z3m3m`), then asserts the result contains `Match{Name: "Daisuushii", IsYakuman: true}` per Yaku Detection — V1 Set.
- [x] 2.2 In `internal/riichi/yaku/yaku.go`, add `func detectDaisuushii(d hand.Decomposition, h hand.Hand, ctx Context) []Match` that returns a yakuman match when `d.Sets()` contains a `MeldTriplet` at each of the four wind bases. Register it in `Detectors()`.

## 3. Shousuushii — three winds plus pair of fourth

- [x] 3.1 In `internal/riichi/yaku/yaku_test.go`, add a failing test `TestDetectShousuushiiThreeWindsAndPair` that builds a winning hand with triplets at three wind bases AND a pair of the fourth wind (e.g. `1z1z1z2z2z2z3z3z3z4z4z3m3m3m`), then asserts the result contains `Match{Name: "Shousuushii", IsYakuman: true}` AND `Daisuushii` is NOT present, per Yaku Detection — V1 Set.
- [x] 3.2 In `internal/riichi/yaku/yaku.go`, add `func detectShousuushii(d hand.Decomposition, h hand.Hand, ctx Context) []Match` that returns a yakuman match when `d.Sets()` contains triplets at exactly three wind bases AND `d.Pair().Base` is the fourth wind. Register it in `Detectors()`.
- [x] 3.3 In `internal/riichi/yaku/yaku.go`, extend `Evaluate` with a "Daisuushii supersedes Shousuushii" cross-cutting rule (mirroring the existing Suuankou-supersedes-Sanankou block): when `Daisuushii` is present in the matches, drop any `Shousuushii` entry before the yakuman-filter step.

## 4. Tsuuiisou — all honors

- [x] 4.1 In `internal/riichi/yaku/yaku_test.go`, add a failing test `TestDetectTsuuiisouAllHonorsStandard` that builds a winning hand of all honors (e.g. `1z1z1z2z2z2z3z3z3z4z4z4z5z5z`) and asserts `Match{Name: "Tsuuiisou", IsYakuman: true}` is present, per Yaku Detection — V1 Set.
- [x] 4.2 In `internal/riichi/yaku/yaku_test.go`, add a failing test `TestDetectTsuuiisouAllHonorsChiitoitsu` that builds the chiitoitsu fixture `1z1z2z2z3z3z4z4z5z5z6z6z7z7z` and asserts `Tsuuiisou` is present, per Yaku Detection — V1 Set.
- [x] 4.3 In `internal/riichi/yaku/yaku.go`, add `func detectTsuuiisou(d hand.Decomposition, h hand.Hand, ctx Context) []Match` that returns a yakuman match when every tile in `h.Concealed` has `IsHonor() == true`. Register it in `Detectors()`.

## 5. Chinroutou — all numeric terminals

- [x] 5.1 In `internal/riichi/yaku/yaku_test.go`, add a failing test `TestDetectChinroutouAllTerminalsStandard` that builds the fixture `1m1m1m9m9m9m1p1p1p9p9p9p1s1s` and asserts `Match{Name: "Chinroutou", IsYakuman: true}` is present, per Yaku Detection — V1 Set.
- [x] 5.2 In `internal/riichi/yaku/yaku.go`, add `func detectChinroutou(d hand.Decomposition, h hand.Hand, ctx Context) []Match` that returns a yakuman match when `d.Form == hand.FormStandard` AND every tile in `h.Concealed` has `IsTerminal() == true` (rank 1 or 9, not an honor). Register it in `Detectors()`.

## 6. Ryuuiisou — all greens

- [x] 6.1 In `internal/riichi/yaku/yaku_test.go`, add a failing test `TestDetectRyuuiisouAllGreens` that builds the fixture `2s2s3s3s4s4s2s3s4s6s6s8s8s8s` (two iipeikou shapes plus 6s pair plus 8s triplet, all greens) and asserts `Match{Name: "Ryuuiisou", IsYakuman: true}` is present, per Yaku Detection — V1 Set.
- [x] 6.2 In `internal/riichi/yaku/yaku.go`, add `func detectRyuuiisou(d hand.Decomposition, h hand.Hand, ctx Context) []Match` that returns a yakuman match when every tile in `h.Concealed` has an ID in the green-tile set `{tile.S2, tile.S3, tile.S4, tile.S6, tile.S8, tile.Hatsu}`. Use a 34-bool lookup or a small `map[uint8]bool` defined as a package-level variable. Register it in `Detectors()`.

## 7. Chuuren poutou — single-suit 1112345678999 + N

- [x] 7.1 In `internal/riichi/yaku/yaku_test.go`, add a failing test `TestDetectChuurenPoutouSingleSuit` that builds the fixture `1m1m1m2m3m4m5m5m6m7m8m9m9m9m` (the 13 base ranks of the man suit plus an extra 5m) with `CalledMelds = nil` and asserts `Match{Name: "Chuuren poutou", IsYakuman: true}` is present, per Yaku Detection — V1 Set.
- [x] 7.2 In `internal/riichi/yaku/yaku_test.go`, add a failing test `TestDetectChuurenPoutouRejectsOpenHand` that uses the same 14-tile bag but sets `h.CalledMelds = [{Kind: hand.CalledPon, BaseID: tile.M1}]`, asserts `Chuuren poutou` is NOT present, per Yaku Detection — V1 Set (concealed-only).
- [x] 7.3 In `internal/riichi/yaku/yaku.go`, add `func detectChuurenPoutou(d hand.Decomposition, h hand.Hand, ctx Context) []Match` that returns a yakuman match when (a) `!h.Open` AND `len(h.CalledMelds) == 0`, (b) every concealed tile belongs to a single numeric suit, AND (c) the per-rank counts within that suit, after subtracting one occurrence of the winning tile's rank, equal `[3, 1, 1, 1, 1, 1, 1, 1, 3]` exactly. Register it in `Detectors()`.

## 8. Verification

- [x] 8.1 Run `go test ./...` from the repository root and confirm all packages pass — especially that no existing yaku test broke from the seven new detectors or the daisuushii-supersedes-shousuushii rule.
- [x] 8.2 Run `golangci-lint run ./...` and resolve any lint issues introduced by the new detectors.
- [x] 8.3 Smoke-test by running `mahjong calc 1m1m1m9m9m9m1p1p1p9p9p9p1s1s` and confirming the output reports `Yaku: Chinroutou (13)` (yakuman, 32000 non-dealer ron).
