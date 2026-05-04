## Why

The human player's `K` key is wired to a working handler (`internal/play/kan_keys.go: handleKan`) that submits the correct engine inputs for all three kan variants:

- Ankan from `AwaitingDiscard{Human}` via `InputDeclareAnkan{TileID}` (lowest-ID 4-of-a-kind).
- Shouminkan from `AwaitingDiscard{Human}` via `InputDeclareShouminkan{TileID}` (lowest-ID concealed tile matching an existing open pon).
- Minkan from `AwaitingClaims` via `InputResolveClaims{Claims: {Human: {Kind: ClaimKan}}}`.

The chankan ron window flows through the existing `R` key in `StateAwaitingChankan` since the engine accepts `InputResolveClaims` with `ClaimRon` there.

What is missing is purely visual and testing:

- The action footer hard-codes `K: Greyed: true` in `FooterKeys`. During the human's discard turn, when ankan or shouminkan is legal, the footer still renders `[K]an` greyed — the player has no signal that pressing K will fire. (`RenderCallFooter` already handles per-state liveness for the claim window correctly.)
- Zero test coverage for `handleKan` in the play layer. Engine tests exist (`internal/game/kan_test.go`), but the play-screen wiring is untested.

## What Changes

- During `AwaitingDiscard{Player: HumanSeat}`, the action footer's `K` SHALL render live when EITHER (a) the human's concealed hand contains a 4-of-a-kind (ankan eligible) OR (b) the human has an open MeldPon AND the matching 4th tile in their concealed hand (shouminkan eligible). When the human is in declared riichi, K SHALL render greyed regardless (the engine rejects kans in riichi for v1; the ack text path handles the message). When neither condition holds, K SHALL render greyed.
- A new helper `humanKanLegal(m Model) bool` SHALL be the single source of truth for the K-live decision. The footer-render code, the tests, and any future smart-AI hint code share this predicate. The helper composes the existing `firstAnkanID` and `firstShouminkanID` helpers in `internal/play/kan_keys.go` and adds the riichi guard.
- Tests covering all four kan code paths in the play layer:
  - Ankan: hand with `4× 1m` plus 9 fillers, state `AwaitingDiscard{Human}` → `humanKanLegal` returns true; pressing K transitions out of `AwaitingDiscard` (engine moves to rinshan flow); `m.ackText` reads `ankan declared`.
  - Shouminkan: hand contains `1m` plus an existing `MeldPon{1m, 1m, 1m}` from a prior call, state `AwaitingDiscard{Human}` → `humanKanLegal` returns true; pressing K transitions to `StateAwaitingChankan{Declarer: Human, UpgradeTile: 1m}`; `m.ackText` reads `shouminkan declared`.
  - Minkan: claims state with discard `5p` and human hand containing `3× 5p` → call window's `[K]an` is live (existing `RenderCallFooter` path); pressing K transitions out of `AwaitingClaims` via `InputResolveClaims{Claim: ClaimKan}`.
  - K-greyed cases: hand has only `3× 1m` (not 4); hand has `1m` but no matching pon; hand is in declared riichi; state is `AwaitingClaims` with no matching tiles. `humanKanLegal` returns false in all cases.
- The action footer rendering (`renderFooter` in `internal/play/play.go`) SHALL be extended to compute the K-key liveness from `humanKanLegal` per render rather than reading the static `Greyed` field for that one key. Other keys (D, R, T, P, C, Spc) remain on the static-greyed-but-functional path; this change is scoped to K only.

## Non-Goals (optional)

- Whole-footer per-state liveness rendering (D/R/T/P/C/Spc all becoming context-aware). The action footer's other action keys are functional but always render as greyed; that visual gap is broader and belongs in a follow-up. This change ships only the K-key live indicator.
- Changing the cursor-based vs. lowest-ID kan-tile selection. The existing handler picks the lowest-ID eligible tile deterministically. Cursor-based selection (where the cursor tile dictates which kan to declare) is a UX refinement deferred to a follow-up if requested.
- Bots declaring kan (`Bot.ShouldKan() bool { return false }` stays false). Smart-AI follow-up.
- Pao / sekinin barai for kan-related yakuman. Separate change.
- Adding a UI affordance for the chankan ron window (the existing R-key in claims state already covers it; verifying via test is in scope).

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `play-screen`: The `Keybinding Map` requirement is updated so the K key documents context-aware live/greyed semantics in the action footer.

## Impact

- Affected specs:
  - `openspec/specs/play-screen/spec.md` (modified): the `Keybinding Map` requirement adds a scenario for the K-key live/greyed transitions.
- Affected code:
  - Modified: `internal/play/play.go` — extend the action-footer render path so the K key's live/greyed style is computed from `humanKanLegal(m)` per render.
  - Modified: `internal/play/kan_keys.go` — add the `humanKanLegal(m Model) bool` helper composing the existing `firstAnkanID` / `firstShouminkanID` predicates with a riichi guard.
  - Modified: `internal/play/play_test.go` — tests for ankan/shouminkan/minkan happy paths and the K-greyed-when-illegal cases.
