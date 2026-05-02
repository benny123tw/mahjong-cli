## Why

The author is a Taiwanese-mahjong player learning Japanese (riichi) mahjong. Riichi differs sharply from Taiwanese mahjong on the rules that bite newcomers — yaku-required wins, furiten, han+fu scoring, chi-from-left only — and these are exactly the rules that benefit from instant feedback. A CLI hand calculator is the smallest deliverable that provides that feedback and, separately, becomes the load-bearing rules engine for a future TUI play mode.

Building the rules engine first (before any TUI) protects the engine's interface from being shaped by UI mocks and gives the project a fully testable core from day one.

## What Changes

- Initialize the Go module (`go.mod`) and project layout
- Add a `riichi` rules-engine package with **zero UI dependencies**
- Implement tile parsing in standard riichi notation: `1m`–`9m`, `1p`–`9p`, `1s`–`9s`, `1z`–`7z`, with `0m`/`0p`/`0s` for red fives
- Implement winning-hand detection covering the three agari shapes (standard 4-sets-and-a-pair, chiitoitsu, kokushi musou)
- Implement shanten and machi (wait) calculation
- Implement detection for the v1 yaku set: riichi, menzen tsumo, pinfu, tanyao, yakuhai (round wind / seat wind / each dragon), iipeikou, toitoi, honitsu, sanshoku doujun, ittsuu
- Extend the v1 yaku set with **Group A — pure-detection standard yaku** that were missed in the original scope: chinitsu (6/5 han), sanankou (2 han, with ron/tsumo distinction on the winning triplet), sanshoku doukou (2/2 han), chanta (2/1 han), junchan (3/2 han), honroutou (2 han), shousangen (2 han plus the two dragon-triplet yakuhai it implies), ryanpeikou (3 han concealed only)
- Implement full fu calculation (not han-only) so the user can practice fu while learning
- Implement final scoring (han + fu → points, including mangan/haneman/baiman/yakuman caps)
- Add a `mahjong calc` cobra subcommand that takes a hand string + optional context flags (seat wind, round wind, riichi declared, tsumo/ron, dora indicator) and prints shanten, machi, yaku list, han, fu, and points
- Add table-driven tests with one fixture per yaku and golden tests for the `calc` CLI

## Non-Goals (optional)

- TUI play mode (deferred to a later change once the engine is stable)
- AI opponents (deferred)
- Trainer aids — machi peek mid-game, furiten warnings, illegal-call greying (deferred; the CLI calculator is the v1 trainer surface)
- Yakuman beyond kokushi musou detection — suuankou, daisangen, chinroutou, etc. are deferred to a follow-up "Group B" change
- Situational/turn-aware yaku — ippatsu, haitei, houtei, rinshan kaihou, chankan, double riichi, tenhou, chiihou — deferred to the TUI play-loop change because they require turn-state context that `mahjong calc` cannot supply
- Kan-related yaku and fu — sankantsu, suukantsu, and kan additions to the fu table — deferred to a follow-up change that introduces kan support across the tile/hand/score packages
- Local-rule yaku — nagashi mangan, renhou, open tanyao toggle (kuitan), etc.
- Replay format and game-history persistence
- Networked or hot-seat multiplayer
- Image-based tile rendering (kitty / sixel / iTerm2 protocols) — out of scope; this change has no visual component beyond CLI text output
- Using `flag` from stdlib instead of cobra — rejected because the project will grow to 3+ subcommands (`play`, `calc`, future `replay`/`rules`); cobra earns its weight from the start

## Capabilities

### New Capabilities

- `hand-calculator`: Parse a riichi mahjong hand from CLI notation, detect winning shapes, identify v1 yaku, compute han + fu + points, and report shanten and machi for tenpai hands.

### Modified Capabilities

(none)

## Impact

- Affected specs: new capability `hand-calculator`
- Affected code:
  - New:
    - go.mod
    - go.sum
    - main.go
    - cmd/root.go
    - cmd/calc.go
    - internal/riichi/tile/tile.go
    - internal/riichi/tile/tile_test.go
    - internal/riichi/tile/parse.go
    - internal/riichi/tile/parse_test.go
    - internal/riichi/hand/hand.go
    - internal/riichi/hand/hand_test.go
    - internal/riichi/hand/shanten.go
    - internal/riichi/hand/shanten_test.go
    - internal/riichi/hand/machi.go
    - internal/riichi/hand/machi_test.go
    - internal/riichi/hand/agari.go
    - internal/riichi/hand/agari_test.go
    - internal/riichi/yaku/yaku.go
    - internal/riichi/yaku/yaku_test.go
    - internal/riichi/score/fu.go
    - internal/riichi/score/fu_test.go
    - internal/riichi/score/score.go
    - internal/riichi/score/score_test.go
    - internal/riichi/calc/calc.go
    - internal/riichi/calc/calc_test.go
    - testdata/calc/golden/.gitkeep
  - Modified:
    - internal/riichi/yaku/yaku.go (extended with the 8 Group A detectors)
    - internal/riichi/yaku/yaku_test.go (extended with Group A fixtures and interaction cases)
    - testdata/calc/golden/ (new golden file covering chinitsu+toitoi+sanankou interaction)
  - Removed: (none)
- Dependencies added: `github.com/spf13/cobra` v1.10.2
