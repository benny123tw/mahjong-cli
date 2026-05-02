## 1. Engine: Riichi state and Riichi Declaration input

- [x] 1.1 Implement Riichi Declaration with combined `InputDiscard{Index, Riichi: bool}` — not a separate `InputDeclareRiichi`: extend `internal/game/state.go` with a `Riichi bool` field on `InputDiscard`. Default zero-value `false` keeps every existing caller of `InputDiscard{Index: N}` unchanged.
- [x] 1.2 In `internal/game/turn.go`, implement Per-Seat Riichi State Lives On Game As Three Parallel Slices: add three parallel slices to `Game`: `riichiDeclared [numSeats]bool`, `ippatsuLive [numSeats]bool`, `doubleRiichi [numSeats]bool`. Add a `scores [numSeats]int` slice initialized to 25000 for each seat in `New()`.
- [x] 1.3 Add a new sentinel error `ErrIllegalRiichi` to `internal/game/turn.go` alongside the existing sentinels.
- [x] 1.4 In `stepFromAwaitingDiscard`, when handling `InputDiscard` with `v.Riichi == true`, validate the four preconditions before applying the discard: (a) `g.IsHandOpen(s.Player) == false`, (b) `g.scores[s.Player] >= 1000`, (c) `g.wall.LiveRemaining() >= 4`, (d) post-discard hand at index `v.Index` removed leaves a 13-tile hand with `hand.Shanten == 0`. Return `ErrIllegalRiichi` on any failure with no state change.
- [x] 1.5 On successful riichi declaration: deduct 1000 from `g.scores[s.Player]`, set `g.riichiDeclared[s.Player] = true`, set `g.ippatsuLive[s.Player] = true`, set `g.doubleRiichi[s.Player] = !g.callsHappened && g.noPriorDiscards()`, then complete the discard transition normally (advance to `StateAwaitingClaims`).

## 2. Engine: Riichi-Restricted Discard

- [x] 2.1 Implement Riichi-Restricted Discard enforced by comparing index against just-drawn tile: in `stepFromAwaitingDiscard`, when the seat already has `riichiDeclared[s.Player] == true` AND `v.Riichi == false` (regular discard from a riichi-declared seat), validate `v.Index == len(g.hands[s.Player]) - 1`. Return `ErrIllegalDiscard` on mismatch with no state change.
- [x] 2.2 In `internal/game/riichi_test.go` (new file), add a test asserting a riichi-declared seat's regular `InputDiscard{Index: 0}` returns `ErrIllegalDiscard`, and `InputDiscard{Index: 13}` succeeds.

## 3. Engine: Ippatsu Window Tracking

- [x] 3.1 Implement Ippatsu Window Tracking close-on-draw: in `stepFromAwaitingDraw`, after the draw transition completes, set `g.ippatsuLive[s.Player] = false` (the seat's own next draw closes their window).
- [x] 3.2 Implement Ippatsu Window Tracking close-on-call: in `stepFromAwaitingClaims`, in both the pon and chi branches (right after `g.callsHappened = true`), iterate over all seats and set `g.ippatsuLive[seat] = false` for any seat with `riichiDeclared[seat] == true`. (A call from anyone breaks ippatsu for everyone in riichi.)
- [x] 3.3 In `contextForWin`, set `ctx.Ippatsu = g.riichiDeclared[winner] && g.ippatsuLive[winner]`. Set `ctx.Riichi = g.riichiDeclared[winner]` and `ctx.DoubleRiichi = g.doubleRiichi[winner]`.
- [x] 3.4 In `internal/game/riichi_test.go`, add a test asserting that after riichi → opponents' draw+discard with no calls → human tsumo, `Ippatsu = true` is set in the calc context (verify by checking the result's yaku list contains ippatsu).
- [x] 3.5 In `internal/game/riichi_test.go`, add a test asserting that after riichi → an opponent calls pon → human tsumo, `Ippatsu = false` (yaku list excludes ippatsu).

## 4. Engine: Double Riichi Detection

- [x] 4.1 Implement Double Riichi Detection reuses existing `noPriorDiscards` and `!callsHappened` logic: confirm task 1.5's `doubleRiichi[s] = !g.callsHappened && g.noPriorDiscards()` runs BEFORE the discard updates `hasDiscarded[s]` so the check sees the pre-discard state. (Re-order if necessary.)
- [x] 4.2 In `internal/game/riichi_test.go`, add a test asserting that the dealer (East) declaring riichi on their first uninterrupted draw sets `doubleRiichi[East] = true`, and a follow-up tsumo passes `DoubleRiichi: true` to `calc.Analyze`.

## 5. Engine: Furiten Query

- [x] 5.1 Implement Furiten Query — Permanent Furiten Implementation Walks The Seat's Own Pond Once Per Query: add `Game.IsFuriten(seat Seat) bool` to `internal/game/turn.go`. Implementation: if `len(g.hands[seat]) != 13`, return false. Compute `m := hand.Machi(hand.Hand{Concealed: g.hands[seat]})`. For each tile `t` in `g.discards[seat]`, if `t.ID` is in `m`, return true. Otherwise return false.
- [x] 5.2 Tests Expand Existing Files Plus One New File: in `internal/game/furiten_test.go` (new file), add three tests covering: machi tile in own pond → true; machi tiles absent from own pond → false; non-tenpai 13-tile hand → false.

## 6. Engine: Human Ron From Claim Window

- [x] 6.1 Implement Human Ron From Claim Window — new sentinel error: add `ErrFuritenRon` to `internal/game/turn.go`.
- [x] 6.2 In `stepFromAwaitingClaims`, in the `ClaimRon` branch, BEFORE building the winning hand and calling `calc.Analyze`, check `if winner == HumanSeat && g.IsFuriten(winner) { return nil, ErrFuritenRon }`. (Bots are not subject to furiten in v1 since bots never ron.)
- [x] 6.3 In `internal/game/riichi_test.go`, add a test asserting the ron path: plant the human into a tenpai shape, drive an opponent's discard of the winning tile, submit `InputResolveClaims{Claims: {Human: ClaimRon}}`, assert state advances to `StateRoundOver{Outcome: OutcomeRon{...}}`.
- [x] 6.4 In `internal/game/riichi_test.go`, add a test asserting furiten ron rejection: plant a tenpai shape, plant the machi tile into the human's own pond, drive an opponent's discard of the winning tile, submit ron, assert `ErrFuritenRon` and state unchanged.

## 7. TUI: Human Riichi Key Binding

- [x] 7.1 Implement Human Riichi Key Binding: in `internal/play/play.go`, replace the existing `r` key handler that sets the "riichi: not implemented" ack with a state-aware dispatch. When state is `StateAwaitingDiscard{Player: HumanSeat}`, call a new `handleRiichi()` method.
- [x] 7.2 Implement `handleRiichi()`: submit `InputDiscard{Index: m.cursor, Riichi: true}`. On `ErrIllegalRiichi`, set `ackText` to a descriptive substring (probe each precondition manually so the message can say "riichi: hand not tenpai" vs "riichi: insufficient funds" vs "riichi: wall has <4 tiles" vs "riichi: hand is open"). On success, clear `ackText`.
- [x] 7.3 In `internal/play/play_test.go`, add a test driving R in `AwaitingDiscard{Human}` with a tenpai post-discard hand at the cursor index, asserting state advances to `AwaitingClaims{Discarder: Human}` and `ackText` is empty.
- [x] 7.4 In `internal/play/play_test.go`, add a test driving R with a NON-tenpai post-discard hand, asserting state is unchanged and `ackText` contains "tenpai".

## 8. TUI: Human Ron Key Binding

- [x] 8.1 Implement Human Ron Key Binding: in the same `r` key dispatch from task 7.1, when state is `StateAwaitingClaims` AND `cs.Discarder != HumanSeat`, call a new `handleRon()` method.
- [x] 8.2 Implement `handleRon()`: build the trial concealed-plus-discard tile slice, run `calc.Analyze` to check for a yaku-bearing winning shape. If nil → set `ackText = "ron: no yaku"`. Else if `m.game.IsFuriten(HumanSeat)` → set `ackText = "ron: furiten"`. Else submit `InputResolveClaims{Claims: {HumanSeat: ClaimRon}}`.
- [x] 8.3 In `internal/play/play_test.go`, add a test driving R in claims window with a yaku-bearing winning hand and no furiten, asserting state advances to `RoundOver{Outcome: Ron}`.
- [x] 8.4 In `internal/play/play_test.go`, add a test driving R with a winning shape but furiten, asserting state is unchanged and `ackText` contains "furiten".

## 9. TUI: Call Window Prompt — live state with furiten suffix

- [x] 9.1 Update Call Window Prompt: render the [R]on key live or greyed based on real-time legality (Furiten Hint Replaces Footer Item Greyed Style). In `RenderCallFooter`, replace the hardcoded `render("[R]on (greyed)", false)` with a live computation. Compute `canRon` = `calc.Analyze(concealed+discard, contextForWin) != nil` AND `!g.IsFuriten(HumanSeat)`. Compute the furiten-suffix case: `furitenBlock` = `calc.Analyze(...) != nil` AND `g.IsFuriten(HumanSeat)`. Render `[R]on` live when canRon, render `[R]on (furiten)` greyed when furitenBlock, render `[R]on` greyed plain otherwise.
- [x] 9.2 Leave `[K]an (greyed)` hardcoded — kan support is deferred. Add a comment in `RenderCallFooter` explaining the asymmetry.
- [x] 9.3 In `internal/play/play_test.go`, add a test asserting `RenderCallFooter` output contains `[R]on` rendered live (no `(furiten)` suffix) when the trial setup gives a yaku-bearing wait with no own-pond machi tile.
- [x] 9.4 In `internal/play/play_test.go`, add a test asserting `RenderCallFooter` output contains `(furiten)` substring when the trial setup gives a yaku-bearing wait with the machi tile planted in own pond.

## 10. Verification

- [x] 10.1 Run `go test ./...` from project root and confirm all suites pass.
- [x] 10.2 Run `golangci-lint run ./...` and confirm 0 issues.
- [x] 10.3 Run `spectra validate add-human-agari` and confirm valid.
- [x] 10.4 Smoke test: `./bin/mahjong play --seed 7` (from earlier scan, this seed has a human chi opportunity around turn 50; chi-then-ron unlikely but the riichi path is exercisable). Press R in `AwaitingDiscard` to trigger riichi flow at least once and observe footer behaviour.
