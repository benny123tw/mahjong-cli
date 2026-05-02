## Why

The human player can pon, chi, and pass during the call window, but cannot win. There is no path to declare ron on a discard, no path to declare riichi, and no furiten enforcement. The footer's `[R]on (greyed)` and `[K]an (greyed)` labels are hardcoded markers reflecting a deferred-from-v1 placeholder. As of seed=42 turn=14 the human can hit a winning shape against an opponent's discard and the only available action is `[Space] Pass`. Since the project is named "riichi mahjong" and the user is learning JP riichi, the missing riichi-and-ron path is the most visible game-feature gap.

A separate motivation: Group C yaku detection (Ippatsu, DoubleRiichi, Tenhou, Chiihou, Haitei, Houtei) is already plumbed end-to-end in the engine and tested, but the riichi-dependent ones (Ippatsu, DoubleRiichi) can never trigger because no caller ever sets `Game.riichi = true`. Wiring riichi declaration unlocks dormant detection paths.

## What Changes

The engine gains a small amount of per-seat riichi state — a bool flag, the turn the riichi was declared (for ippatsu's "no calls between declaration and own next draw" check), and a flag tracking whether the ippatsu window is still open. `Game.contextForWin` surfaces these flags to `calc.Analyze` for scoring.

The engine gains a `Game.IsFuriten(seat) bool` query. Furiten is permanent (whole-round) when the seat has any machi tile in their own pond after declaring riichi, and temporary (until next draw) when an opponent discarded a machi tile since the seat's last draw and the seat passed on it. v1 implements permanent furiten; temporary furiten ships when bot ron lands (add-smart-ai).

The engine accepts a new input `InputDeclareRiichi` valid only in `StateAwaitingDiscard{Player}` when the seat: is concealed (no called melds), has ≥1000 points, the live wall has ≥4 tiles remaining, and is tenpai when considering each possible discard. The seat selects which tile to discard along with the riichi declaration; the resolved transition deducts the 1000-point riichi deposit, marks the seat as riichi-declared, opens the ippatsu window, and proceeds to the normal post-discard `StateAwaitingClaims`. After riichi is declared, subsequent `InputDiscard` from that seat is restricted: only the just-drawn tile may be discarded (no choosing). Tsumo and kan-of-just-drawn-tile remain legal.

The TUI wires two new keys for the human: `r` while in `StateAwaitingClaims{Discarder: !=Human}` submits `InputResolveClaims{Claims: {Human: ClaimRon}}`; `r` while in `StateAwaitingDiscard{Human}` submits `InputDeclareRiichi` with the cursor's tile index and proceeds. `RenderCallFooter` evaluates `canRon` (running `calc.Analyze` on `concealed + discard` returning non-nil AND `IsFuriten == false`) and `canRiichi` (running tenpai + funds + wall checks) live, and renders `[R]on` and `[R]iichi` accordingly. The hardcoded `(greyed)` suffixes go away; the visual treatment uses the same liveKeyStyle / greyedKeyStyle pattern as `[P]on` / `[C]hi`.

## Non-Goals

- Bot riichi declaration. Bots stay non-riichi-declaring in this change (deferred to add-smart-ai). The riichi state machine works for any seat; bots simply never submit `InputDeclareRiichi`.
- Bot ron from the claim window. Bots auto-pass today; smart bot ron lands in add-smart-ai.
- Pao / sekinin-barai liability rules (the rule that the discarder of a 3rd tile completing a yakuhai-pon-set into a yakuman pays double). Standard ron payout only.
- Riichi confirmation modal. Single keystroke commits the declaration; the player can avoid it by simply pressing `d` for a normal discard instead.
- Kan after riichi. Kan support is deferred to add-kan-support; this change explicitly disallows ankan/added-kan after riichi (which is normally legal but only for kans that don't change the wait — out of v1 scope).
- Temporary furiten across multiple opponent discards (the "missed ron makes you furiten until your next draw" rule). v1 implements only permanent furiten (machi tile in own pond). Temporary furiten lands when bot ron is wired in add-smart-ai.
- Yakuman from riichi-only situations (e.g., chiihou is already detected; renhou — winning by ron on first uninterrupted intake — stays out of v1, debated rule anyway).

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `game-loop`: adds the riichi state machine, riichi-restricted-discard rule, ippatsu-window tracking, and the human-side ron path through `InputResolveClaims{ClaimRon}` for `HumanSeat`. Adds `Game.IsFuriten(seat)` query.
- `play-screen`: wires the `r` key in two states (claim window → ron, discard state → riichi), updates `RenderCallFooter` to compute `canRon` / `canRiichi` live (no hardcoded greyed labels), and adds a furiten hint string when ron is illegal because of furiten.

## Impact

- Affected specs: modified capability `game-loop` (riichi state, furiten, riichi-restricted-discard, human ron); modified capability `play-screen` (R-key handlers, footer logic, furiten hint).
- Affected code:
  - Modified:
    - internal/game/turn.go (per-seat riichi state, ippatsu-window tracking, IsFuriten implementation, InputDeclareRiichi handler, riichi-restricted-discard enforcement)
    - internal/game/state.go (InputDeclareRiichi type, optional InputDiscard.Riichi field for combined discard+riichi)
    - internal/game/golden_test.go (golden file regenerates if any path changes events; if not, no regen needed)
    - internal/play/play.go (handleRon, handleRiichi, RenderCallFooter rewrite, riichi state in footer of normal discard window, furiten hint)
    - internal/play/play_test.go (tests for ron path, riichi declaration, furiten greys ron, double-riichi flag set)
  - New:
    - internal/game/furiten.go (IsFuriten implementation if it grows beyond a few lines; otherwise live in turn.go)
    - internal/game/riichi_test.go (focused tests for riichi state machine: declaration legal/illegal, ippatsu window open/close, double-riichi turn-1 detection)
  - Removed:
    - (none — `[K]an (greyed)` placeholder stays since kan is still deferred)
