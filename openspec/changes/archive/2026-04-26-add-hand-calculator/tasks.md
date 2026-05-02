## 1. Project bootstrap

- [x] 1.1 Initialize Go module `github.com/benny123tw/mahjong-cli` and add `github.com/spf13/cobra` v1.10.2 dependency (engine-first sequencing, no TUI in this change)
- [x] 1.2 Establish the package layout under `internal/riichi/` with empty `tile`, `hand`, `yaku`, `score`, `calc` sub-packages and a top-level `cmd/` directory for cobra entrypoints

## 2. Tile model

- [x] 2.1 Implement the Tile representation as `uint8` ID with red-five flag — canonical ordering, suit/rank accessors, terminal/honor predicates
- [x] 2.2 Implement Tile Notation Parsing for hand strings (`0m`/`0p`/`0s` red fives, `1z`–`7z` honors), rejecting invalid codes, hands outside 13–14 tiles, and more than 4 copies of any tile
- [x] 2.3 Add unit tests for Tile Notation Parsing covering the example tile-mapping table, invalid codes, oversize hands, and five-copy rejections

## 3. Winning shape and waits

- [x] 3.1 Implement Winning-Hand Detection using recursive set-extraction for standard agari, direct check for chiitoitsu and kokushi
- [x] 3.2 Implement Shanten and Machi Calculation for 13-tile concealed hands, reusing the decomposition machinery from 3.1
- [x] 3.3 Add unit tests covering all Winning-Hand Detection scenarios (standard, chiitoitsu, kokushi, chiitoitsu-fallthrough, non-winning) and Shanten and Machi Calculation across the example-table shapes

## 4. Yaku detection

- [x] 4.1 Implement Yaku Detection — V1 Set with Yaku as independent detector functions: riichi, menzen tsumo, pinfu, tanyao, yakuhai (round wind / seat wind / each dragon), iipeikou, toitoi, honitsu, sanshoku doujun, ittsuu — including open/concealed han downgrades and pinfu shape gating
- [x] 4.2 Add one test fixture per yaku plus interaction fixtures: pinfu+tsumo, iipeikou rejected when open, honitsu/sanshoku/ittsuu open-hand downgrades, yakuhai by wind/dragon, mutual exclusion of iipeikou with chiitoitsu

## 5. Fu and scoring

- [x] 5.1 Implement Fu Calculation — Fu computed from the chosen decomposition, not the raw hand — covering every component in the fu table, pinfu-tsumo flat 20, chiitoitsu flat 25, standard round-up to nearest 10, and kuipinfu
- [x] 5.2 Implement Final Score Calculation including dealer 1.5x payout and mangan / haneman / baiman / sanbaiman / kazoe-yakuman tiers, with true-yakuman capping non-yakuman han
- [x] 5.3 Implement Decomposition Selection — enumerate valid winning decompositions, pick the highest-scoring one, deterministic lexicographic tie-break
- [x] 5.4 Add fu and scoring fixtures: pinfu-tsumo, chiitoitsu, kanchan/penchan/tanki waits, concealed terminal triplets, open-hand kuipinfu, 32 → 40 rounding, dealer-vs-non-dealer payout boundaries, kazoe yakuman

## 6. CLI integration

- [x] 6.1 Wire the CLI surface — implement the CLI Command Surface for `mahjong calc <hand>` using cobra root + calc subcommand with all context flags (`--seat`, `--round`, `--riichi`, `--tsumo`, `--dora`, `--uradora`) and the structured output format (shanten line, yaku list, fu breakdown, points line)
- [x] 6.2 Add golden tests for `mahjong calc` per the Test strategy — golden files under `testdata/calc/golden/`, updatable via `go test -update` — covering successful winning analysis, tenpai-only output, and invalid input

## 7. Group A — extended yaku detection

- [x] 7.1 Extend Yaku Detection — V1 Set with composition-based Group A detectors per the v1 yaku set scoped to detection-only standard yaku design decision: chinitsu (6/5 han, single numeric suit no honors), honroutou (2/2 han, only terminals and honors), chanta (2/1 han, every set and pair contains a terminal or honor), junchan (3/2 han, every set and pair contains a terminal AND no honor anywhere) — including the chanta vs junchan supersession rule when honors are absent
- [x] 7.2 Extend Yaku Detection — V1 Set with meld-shape Group A detectors: sanankou (2 han, with the shanpon-ron exclusion that downgrades the winning triplet to "open" for sanankou-counting purposes), sanshoku doukou (2 han, same-rank triplets across all three suits), shousangen (2 han, two dragon triplets plus a dragon pair, on top of the two yakuhai han the dragon triplets already contribute), ryanpeikou (3 han concealed only, suppressing iipeikou for the same decomposition)
- [x] 7.3 Add fixtures per Group A yaku and interaction cases: chinitsu rejecting honor-bearing hands while honitsu still matches them, sanankou ron-vs-tsumo distinction on a shanpon-completion hand, junchan vs chanta supersession when no honors are present, shousangen with each of the three dragons as the pair, ryanpeikou suppressing iipeikou on a hand admitting both readings, honroutou implying toitoi or chiitoitsu form
- [x] 7.4 Add a CLI golden test for the smoke-test hand `1m1m1m4m4m4m7m7m7m9m9m9m5m5m` so the chinitsu+toitoi+sanankou stack appears in regression coverage
