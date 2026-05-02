## MODIFIED Requirements

### Requirement: Call Window Prompt

The system SHALL render a call-window prompt in the action footer whenever the underlying game-loop state is `AwaitingClaims` and the human player has at least one legal claim available. Each call key SHALL be live or greyed based on real-time legality checks against the human's hand and game state. The hardcoded `[R]on (greyed)` placeholder SHALL be replaced with state-derived rendering: live `[R]on` when `calc.Analyze` returns non-nil for `concealed + discard` AND `Game.IsFuriten(HumanSeat)` is false; greyed `[R]on (furiten)` when the hand would otherwise win but furiten blocks it; greyed `[R]on` (no suffix) when the hand simply does not win. `[K]an (greyed)` SHALL remain a hardcoded placeholder until kan support lands. Pressing `Space` SHALL submit a pass and SHALL transition the state machine via the no-claim path. Greyed keys SHALL NOT advance state. The prompt SHALL NOT enforce a real-time timeout — the player SHALL be allowed unbounded wall-clock thinking time.

#### Scenario: Call window appears after opponent discard with legal pon

- **GIVEN** the underlying game state is `AwaitingClaims{Discard: 5p, Discarder: West}` and the human player has two 5p (legal pon)
- **WHEN** the View is rendered
- **THEN** the action footer shows `[P]on` rendered live
- **AND** pressing `P` advances state to a discard step for the human player with the new pon meld registered

#### Scenario: Live [R]on when ron is legal

- **GIVEN** the underlying game state is `AwaitingClaims{Discard: 5p, Discarder: East}`
- **AND** `calc.Analyze` on the human's `concealed + 5p` returns non-nil
- **AND** `Game.IsFuriten(HumanSeat) == false`
- **WHEN** the View is rendered
- **THEN** the action footer shows `[R]on` rendered live with no `(furiten)` suffix
- **AND** pressing `R` advances state to `RoundOver{Outcome: OutcomeRon{...}}`

#### Scenario: [R]on greyed with furiten suffix

- **GIVEN** the underlying game state is `AwaitingClaims{Discard: 5p, Discarder: East}`
- **AND** `calc.Analyze` on the human's `concealed + 5p` returns non-nil
- **AND** `Game.IsFuriten(HumanSeat) == true`
- **WHEN** the View is rendered
- **THEN** the action footer shows `[R]on (furiten)` rendered greyed
- **AND** pressing `R` does not change state and sets `ackText` to a substring containing "furiten"

#### Scenario: [R]on greyed without suffix when no winning shape

- **GIVEN** the underlying game state is `AwaitingClaims{Discard: 5p, Discarder: East}`
- **AND** `calc.Analyze` on the human's `concealed + 5p` returns nil
- **WHEN** the View is rendered
- **THEN** the action footer shows `[R]on` rendered greyed with no suffix
- **AND** pressing `R` does not change state

#### Scenario: Pass advances state with no claim

- **GIVEN** a call window is active and the human player has no winning hand and no other legal claim
- **WHEN** the human presses `Space`
- **THEN** the human's pass is recorded and the state machine resolves the claim window with no winner from the human's side

#### Scenario: Greyed keys do not mutate state

- **GIVEN** a call window is active where chi is not legal (discarder is not kamicha)
- **WHEN** the human presses `C`
- **THEN** state does not change and a brief footer feedback indicates the key is illegal

---

## ADDED Requirements

### Requirement: Human Riichi Key Binding

The system SHALL bind the `r` key in `StateAwaitingDiscard{Player: HumanSeat}` to riichi declaration: when pressed, submit `InputDiscard{Index: cursor, Riichi: true}` to the engine. When the engine returns `ErrIllegalRiichi`, the TUI SHALL set `ackText` to a description of which precondition failed (e.g., "riichi: hand not tenpai", "riichi: insufficient funds", "riichi: wall has <4 tiles", "riichi: hand is open"). When the engine accepts the riichi, the TUI SHALL clear `ackText` and rely on the new state (advanced to `StateAwaitingClaims`) to drive subsequent rendering.

#### Scenario: R in discard state declares riichi when legal

- **GIVEN** the human is in `StateAwaitingDiscard{Player: HumanSeat}` with cursor at index 13 and a tenpai post-discard hand
- **WHEN** the user presses `r`
- **THEN** the underlying game state advances to `StateAwaitingClaims{Discarder: HumanSeat}`
- **AND** the TUI's `ackText` is empty

#### Scenario: R rejected with descriptive ackText when illegal

- **GIVEN** the human is in `StateAwaitingDiscard{Player: HumanSeat}` and the post-discard hand at cursor index 5 is NOT tenpai
- **WHEN** the user presses `r`
- **THEN** game state is unchanged
- **AND** the TUI's `ackText` contains a substring describing the failure (e.g., "tenpai", "riichi")

---

### Requirement: Human Ron Key Binding

The system SHALL bind the `r` key in `StateAwaitingClaims{Discarder: !=HumanSeat}` to ron declaration: when pressed AND `calc.Analyze` on `concealed + discard` returns non-nil AND `Game.IsFuriten(HumanSeat)` is false, the TUI SHALL submit `InputResolveClaims{Claims: {HumanSeat: ClaimRon}}` to the engine. When pressed AND ron is illegal, the TUI SHALL set `ackText` to either "ron: no yaku" (when `calc.Analyze` returns nil) or "ron: furiten" (when furiten check fails), and leave game state unchanged.

#### Scenario: R in claim window declares ron when legal

- **GIVEN** the human is in `StateAwaitingClaims{Discard: 5p, Discarder: East}`
- **AND** the hand wins yakufully on 5p with no furiten
- **WHEN** the user presses `r`
- **THEN** the game advances to `StateRoundOver{Outcome: OutcomeRon{...}}`

#### Scenario: R rejected with "no yaku" ackText

- **GIVEN** the human is in `StateAwaitingClaims{Discard: 5p, Discarder: East}`
- **AND** the hand completes a winning shape on 5p but has no yaku
- **WHEN** the user presses `r`
- **THEN** game state is unchanged
- **AND** the TUI's `ackText` contains "no yaku"

#### Scenario: R rejected with "furiten" ackText

- **GIVEN** the human is in `StateAwaitingClaims{Discard: 5p, Discarder: East}`
- **AND** the hand wins yakufully on 5p
- **AND** the human's own pond contains 5p (permanent furiten)
- **WHEN** the user presses `r`
- **THEN** game state is unchanged
- **AND** the TUI's `ackText` contains "furiten"
