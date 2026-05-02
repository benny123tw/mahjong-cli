## ADDED Requirements

### Requirement: Match-Bound Model

The play-screen `Model` SHALL hold a `*game.Match` rather than a `*game.Game` so multi-hand state — scores, dealer, round, hand index, honba, riichi sticks — flows through to rendering and transitions. The `NewWithMatch(renderer Renderer, m *game.Match) Model` constructor SHALL be the canonical entry point for the `mahjong play` CLI; `NewWithGame(renderer, g)` MAY remain for tests that need to drive a single round directly. `Model.GameState()` SHALL delegate to `m.match.CurrentGame().State()` so existing per-state rendering paths continue to work unmodified.

#### Scenario: Status bar reflects live match state

- **GIVEN** a model bound to a `*game.Match` at East 2 (handIndex = 1), honba = 1, riichi sticks = 1, scores = `[24000, 25500, 25500, 25000]`
- **WHEN** the model renders the status row
- **THEN** the row displays the round name "East 2", honba 1, riichi pool 1, and each seat's current score (the hardcoded "East 1 · Honba 0 · Score 25000" string SHALL NOT appear)

---

### Requirement: End-of-Hand Acknowledgement

When the active hand reaches `StateRoundOver`, the model SHALL invoke `Match.AdvanceFromOutcome` once, capture the returned `TransitionResult`, and store it as a pending acknowledgement. While an acknowledgement is pending, the model SHALL render an end-of-hand summary panel showing the outcome (ron/tsumo/ryuukyoku), per-seat point deltas, post-payout totals, and (if renchan) the new honba count. The next hand's `*Game` SHALL NOT begin processing inputs until the player presses any key, at which point the model clears the pending acknowledgement and resumes normal play with the new hand's state machine. If the `TransitionResult` includes a populated `MatchOutcome`, the acknowledgement SHALL transition to the end-of-match standings screen instead of the next hand.

#### Scenario: Ron at East 1 shows ack panel and waits for keypress

- **GIVEN** a model bound to a match at East 1, dealer = `SeatEast`, with the active game at `StateRoundOver{Outcome: OutcomeRon{Winner: SeatSouth, Loser: SeatEast, Result: ...}}`
- **WHEN** the model receives any tea.Msg
- **THEN** `Match.AdvanceFromOutcome` is invoked exactly once
- **AND** the model renders a panel showing "South ron from East" plus the per-seat deltas
- **AND** the panel remains visible until any key is pressed

#### Scenario: Keypress on ack panel advances to the next hand

- **GIVEN** the model is showing the end-of-hand ack panel for a non-renchan ron
- **WHEN** the player presses any key
- **THEN** the ack panel clears
- **AND** the model's underlying `*Game` is the new East 2 hand (dealer rotated, handIndex incremented)
- **AND** rendering resumes the normal play layout

---

### Requirement: End-of-Match Standings Screen

When `Match.IsFinished()` returns true, the model SHALL render a standings screen listing all four seats sorted by final score descending, plus the match-end reason ("hanchan-complete" or "tobi: <seat>"). The screen SHALL accept the `q` key to quit and SHALL ignore all other inputs (no further hands to play). The standings screen SHALL replace the normal play layout entirely (no status bar, no pond zones, no hand row).

#### Scenario: Standings shown on hanchan completion

- **GIVEN** a match that has just finished South 4 with non-renchan outcome and `Match.FinalOutcome().Reason == "hanchan-complete"`, scores `[26500, 24500, 27500, 21500]`
- **WHEN** the model renders
- **THEN** the layout shows four rows in descending order: West 27500, East 26500, South 24500, North 21500
- **AND** the reason "hanchan-complete" is displayed
- **AND** pressing `q` quits the program

#### Scenario: Standings shown on tobi

- **GIVEN** a match that ended due to tobi with `Match.FinalOutcome().Reason == "tobi"`, `BustSeat = SeatNorth`
- **WHEN** the model renders
- **THEN** the standings display the four seats with their final scores
- **AND** the reason "tobi: North" (or equivalent) is shown
