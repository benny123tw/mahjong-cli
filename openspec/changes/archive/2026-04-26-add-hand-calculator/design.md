## Context

The project is an empty Go workspace. This change is the first one and bootstraps the entire codebase: `go.mod`, package layout, the riichi rules engine, and the `mahjong calc` CLI.

The author's goal is to learn Japanese (riichi) mahjong from a Taiwanese-mahjong background. A hand calculator gives instant feedback on yaku/fu/score from any hand string and is the smallest deliverable that already provides learning value. Beyond v1, the same engine will back a TUI play mode and trainer aids (machi peek, furiten warnings) — those are out of scope for this change but constrain the engine's interface: it must be callable from both a CLI command and a future event-driven TUI without refactor.

The defining architectural rule for this change: **the engine has zero UI dependencies**, and `cmd/calc.go` is a thin adapter that parses flags, calls the engine, and prints the result. This is enforced by the package layout — engine code lives under `internal/riichi/` and may not import `cmd/`, `bubbletea`, or `lipgloss`.

## Goals / Non-Goals

**Goals:**

- Provide a complete riichi hand evaluator (shanten, machi, yaku, fu, score) usable from `mahjong calc <hand-string>`
- Establish a package layout that the future TUI play mode can import without modification
- Cover the v1 yaku set with one test fixture per yaku, plus golden tests for the `calc` command
- Compute full fu (not han-only), so the user can practice fu while learning
- Support red fives (akadora) and dora indicators

**Non-Goals:**

- TUI play mode, AI opponents, trainer-aid UI — deferred to follow-up changes
- Yakuman beyond kokushi musou (suuankou, daisangen, chinroutou, etc.) — deferred
- Local-rule yaku (nagashi mangan, renhou, kuitan toggle)
- Replay format, persistence, multiplayer
- Image-based tile rendering (kitty / sixel / iTerm2)
- Performance optimization beyond what is needed for sub-second `calc` response on a single hand — no benchmarks, no parallelism

## Decisions

### Engine-first sequencing, no TUI in this change

The TUI is deliberately excluded from v1. Rationale: a UI built against mocked engine data tends to bully the engine's interface into a UI-friendly shape rather than a domain-honest one. By shipping the engine alone (with `mahjong calc` as its only caller), the engine's interface is shaped by the actual problem — "given a hand and context, produce an analysis" — and the future TUI inherits a stable, tested core.

Alternatives considered:
- **TUI-first with mocked engine**: rejected; tends to require refactoring both halves once real data flows.
- **Full game loop (engine + TUI + AI) in one change**: rejected; >15 tasks, multi-week scope, high risk of partial completion.

### Package layout under `internal/riichi/`

```
cmd/                        cobra entrypoints (root.go, calc.go)
internal/riichi/
  tile/                     Tile type, parsing, ordering, akadora flag
  hand/                     Hand struct, shanten, machi, agari (winning-shape detection)
  yaku/                     Yaku detectors, one per yaku, returning han
  score/                    Fu calculator + final score table
  calc/                     Top-level orchestrator: parse → analyze → result struct
testdata/calc/golden/       Golden output files for calc CLI tests
```

Each sub-package has a narrow responsibility and depends only on packages to its left in the dependency graph: `tile → hand → yaku → score → calc`. No cycles. The engine is `internal/` so external Go consumers cannot accidentally take a dependency on it before the API is stable.

Alternatives considered:
- **Single `riichi` package**: rejected; the file count would be high and concerns would tangle.
- **Public `pkg/riichi/`**: rejected; premature commitment to a public API while the engine is still settling.

### Tile representation as `uint8` ID with red-five flag

A `Tile` is a `uint8` ID `0..33` covering the 34 unique tile values (9 man + 9 pin + 9 sou + 7 honors), plus a separate `bool` `Red` field for akadora. Rationale: integer IDs make hand counts trivial (`[34]int8` array), make sorting and comparison cheap, and give yaku detectors clean numeric predicates ("all tiles in `[1..7]` ∪ `[10..16]` ∪ `[19..25]`" for tanyao). Red fives are a scoring concern, not an identity concern (a red 5p plays as a 5p in every shape rule), so `Red` rides alongside the ID rather than expanding the ID space.

Alternatives considered:
- **String-based tiles (`"5p"`)**: rejected; allocation overhead, weak typing, cumbersome comparisons.
- **Struct `{Suit, Rank}`**: rejected; ID arithmetic is more convenient for shanten/agari.
- **Encoding red into the ID**: rejected; would force every shape predicate to normalize.

### Recursive set-extraction for standard agari, direct check for chiitoitsu and kokushi

Standard winning shape (4 sets + pair) uses a recursive decomposition: pick a pair candidate, then greedily extract sequences (chii) and triplets (pon) from the lowest remaining tile. The recursion is bounded (a hand has 14 tiles; depth ≤ 5) and fast enough without memoization for v1. Chiitoitsu and kokushi musou are checked by direct shape predicates first; if neither matches, fall through to the recursive standard check.

Decomposition ambiguity (a hand admitting multiple valid 4-sets-and-a-pair interpretations) is resolved by **picking the decomposition that yields the highest final score (han + fu, after rounding to the standard score table)** — this is the standard riichi rule and matters for hands like `1112345678999m` where two interpretations give different yaku.

Alternatives considered:
- **Bitmask / dynamic-programming shanten**: faster but unnecessary for a single-hand calculator.
- **Pre-computed lookup table for all hand shapes**: tens of MB, overkill.

### Yaku as independent detector functions

Each yaku is a function `Detect(d Decomposition, ctx Context) (matched bool, han int)`. The orchestrator runs every detector against every valid decomposition, then applies yaku interaction rules: pinfu requires a specific shape (all sequences, ryanmen wait, non-yakuhai pair); iipeikou is concealed-only; honitsu and tanyao share no tiles so cannot conflict; etc. Yakuman wins zero out non-yakuman han.

This shape lets new yaku be added in follow-up changes without touching unrelated detectors. Each detector is testable in isolation.

### v1 yaku set scoped to detection-only standard yaku

The v1 set is everything that can be decided from `(Decomposition, Hand, Context)` alone — no turn state, no kan support. Concretely, eighteen yaku in two waves: the original ten (riichi, menzen tsumo, pinfu, tanyao, yakuhai, iipeikou, toitoi, honitsu, sanshoku doujun, ittsuu) plus **Group A — pure-detection additions** (chinitsu, sanankou, sanshoku doukou, chanta, junchan, honroutou, shousangen, ryanpeikou).

The Group A additions are uncovered after the initial implementation by smoke-testing — chinitsu in particular is too common to omit (a single-suit toitoi hand reporting only 2 han is misleading for a study tool). The set is closed under "what `mahjong calc` can decide." Anything that needs more context lives in a different change.

Sanankou requires special handling: the rule counts only triplets that were concealed at the moment of agari. If the winning tile completes a triplet via shanpon-ron, that triplet is treated as open (because the discarded tile came from an opponent), so a hand with three concealed triplets and one shanpon-ron triplet has only two concealed triplets and no sanankou. The detector therefore reads `Hand.IsTsumo` plus the winning tile's position in the decomposition. On tsumo, all four-triplet hands would be suuankou yakuman — explicitly deferred to Group B.

Ryanpeikou supersedes iipeikou: when both could match (a hand with two distinct iipeikou shapes), the detector emits ryanpeikou and the iipeikou detector is suppressed for that decomposition. Both remain concealed-only.

Alternatives considered:
- **Bundle Group A and Group B (yakuman set) into one change**: rejected; Group B introduces a yakuman tier display convention and "what counts as concealed for suuankou" rules that don't belong tangled with Group A's straightforward detection extensions.
- **Defer Group A to a follow-up too, leaving v1 at ten yaku**: rejected once chinitsu's absence was discovered; the calculator becomes misleading for common single-suit hands without it.

Deferral boundaries (recorded explicitly so future architecture reviews don't re-litigate):
- **Group B — detectable yakuman** (daisangen, suuankou, shousuushi, daisuushi, tsuuiisou, ryuuisou, chinroutou, chuuren poutou): own change; needs yakuman-tier presentation rules.
- **Group C — situational/turn-aware** (ippatsu, haitei, houtei, rinshan, chankan, double riichi, tenhou, chiihou): bundled with the TUI play-loop change because they require state `mahjong calc` cannot supply.
- **Group D — kan-related** (sankantsu, suukantsu, plus kan additions to the fu table): bundled with the change that adds kan support across tile/hand/score.
- **Group E — local rules** (nagashi mangan, renhou, daisharin, multiple yakuman): out of scope unless explicitly requested.

### Fu computed from the chosen decomposition, not the raw hand

Fu depends on *which* decomposition you pick (an `4p5p6p` interpreted as a sequence is 0 fu, but the same tiles read as a `4p5p` ryanmen wait inside another shape are different). Fu is therefore computed against the same `Decomposition` the yaku detectors saw. The fu table is a straightforward port of the standard riichi rules: base 20, +2/+4/+8 per closed triplet/kan, +2 for tanki/kanchan/penchan waits, +2 yakuhai pair, +10 menzen ron, +2 tsumo (except pinfu-tsumo which is 20 flat), +2 open hand with no other fu (kuipinfu).

The famously-confusing **pinfu + tsumo** case (20 fu flat, no +2 tsumo bonus) gets an explicit test fixture.

### CLI surface

```
mahjong calc <hand-string> [flags]

Flags:
  --seat E|S|W|N         Seat wind (default S — non-dealer)
  --round E|S            Round wind (default E)
  --riichi               Player declared riichi
  --tsumo                Win by tsumo (default: ron)
  --dora <tile>          Dora indicator (repeatable)
  --uradora <tile>       Ura-dora indicator (repeatable; only counted with --riichi)
```

The hand string concatenates tiles in any order: `1m2m3m4p5p0p7s8s9s1z1z2z2z2z` (14 tiles for a winning hand, 13 for tenpai-only analysis). Output is a structured text block: shanten, machi (if tenpai), yaku list with han, fu breakdown, final points.

### Test strategy

- **Unit tests** colocate with each package; one fixture per yaku in `yaku_test.go`; fu edge cases (especially pinfu-tsumo, kanchan, tanki, open hand with kuipinfu) in `fu_test.go`.
- **Golden tests** for `mahjong calc`: input hand strings + expected formatted output, stored under `testdata/calc/golden/`. Update with `go test -update`.
- No mocks. Engine is pure; CLI shells out to engine. Tests exercise real code paths.

## Risks / Trade-offs

- **Yaku-interaction bugs** (pinfu requires no fu other than menzen-ron 10; iipeikou is concealed-only and cannot coexist with chiitoitsu in a single decomposition). → Mitigation: one fixture per yaku plus dedicated interaction fixtures (pinfu+tsumo, iipeikou+chiitoitsu rejection, sanshoku+ittsuu interaction).
- **Decomposition ambiguity** producing wrong scores for hands like `1112345678999m`. → Mitigation: orchestrator enumerates all valid decompositions and picks the highest-scoring one; explicit fixture for the canonical ambiguous hand.
- **Akadora confusion** — red fives count as +1 han but participate in shape rules as 5s. → Mitigation: red-five flag separate from tile ID; shape predicates ignore it; scoring code reads it.
- **Scope creep into yakuman / local rules** mid-implementation. → Mitigation: Non-Goals are explicit; yakuman beyond kokushi triggers a follow-up change, not an addition here.
- **Cobra adds dependency weight for one subcommand**. → Accepted; the project will grow to `play` and likely `replay` within a few changes, and switching from `flag` to cobra later is annoying.
- **Engine-first means no visible progress for a week or two**. → Accepted; the `mahjong calc` CLI is the visible deliverable and is genuinely useful on its own as a study tool.
