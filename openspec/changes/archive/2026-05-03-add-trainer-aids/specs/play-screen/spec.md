## ADDED Requirements

### Requirement: Trainer Aids In The Action Footer

The system SHALL surface three learning aids in the action footer to help a player who is learning JP riichi see the game's wait, furiten, and call-legality state without changing engine behaviour. Two of the three aids ship in this requirement (machi peek toggle and the standalone furiten badge); the third (illegal-call greying inside the call window) is already provided by the existing `Call Window Prompt` requirement and is referenced here for completeness only.

The system SHALL bind the `?` key (KeyPress code `'?'`) as a live binding (no longer greyed in `FooterKeys`). Pressing `?` SHALL toggle a TUI-only `peekVisible bool` flag on the Model. Toggling SHALL NOT call any `m.game.Step` and SHALL NOT mutate underlying game state.

When `peekVisible` is true, the action footer SHALL render an extra line directly below the action keys formatted as one of:

- `Wait: <id1> <id2> ...` — when the cached `peekShanten` equals 0 AND `len(peekMachi) > 0`. Each tile ID is rendered via `tile.Tile{ID: id}.String()` separated by single spaces.
- `Wait: (not tenpai)` — when `peekShanten != 0` OR `peekMachi` is empty.

`peekVisible` SHALL be reset to `false` whenever the existing peek-cache reset sites fire (after a discard, after the bot-tick handler clears the cache, and at the pending-transition handler). The peek auto-hides on state change without requiring a second `?` press.

The action footer SHALL render a `[FURITEN]` badge when (a) the current state is `StateAwaitingDraw{Player: HumanSeat}` OR `StateAwaitingDiscard{Player: HumanSeat}` (the human's own turn cycle, NOT the call window), (b) the human's concealed hand is at tenpai (`hand.Shanten(humanHand) == 0`), AND (c) `Game.IsFuriten(HumanSeat)` returns true. The badge SHALL render in red via a `furitenBadgeStyle` lipgloss style in the Unicode renderer; the ASCII renderer SHALL render the badge as the literal string `(furiten)` with no color codes.

#### Scenario: Question mark binds peek visibility toggle

- **GIVEN** the human is in `StateAwaitingDiscard{Player: HumanSeat}` with `peekVisible == false`
- **WHEN** the human presses `?`
- **THEN** `peekVisible` becomes true
- **AND** `m.game.State()` is unchanged
- **WHEN** the human presses `?` again
- **THEN** `peekVisible` becomes false

#### Scenario: Wait line shows machi IDs when tenpai

- **GIVEN** the human's 13-tile hand is `1m2m3m4p5p6p7s8s9s2z2z3z3z` (tenpai on `4p` and `7p`)
- **AND** `peekVisible == true`
- **WHEN** the action footer renders
- **THEN** the rendered output contains a line "Wait: 4p 7p"

##### Example: peek of a non-tenpai hand

- **GIVEN** the human's hand has `peekShanten == 1`
- **AND** `peekVisible == true`
- **WHEN** the action footer renders
- **THEN** the rendered output contains the line "Wait: (not tenpai)"

#### Scenario: Peek auto-clears on discard

- **GIVEN** `peekVisible == true` and the human is in `StateAwaitingDiscard`
- **WHEN** the human discards a tile
- **THEN** `peekVisible` becomes false
- **AND** the rendered footer no longer shows the wait line

#### Scenario: Furiten badge appears during human's turn when tenpai and furiten

- **GIVEN** `Game.IsFuriten(HumanSeat)` is true and the human's concealed hand has `Shanten == 0`
- **AND** the current state is `StateAwaitingDraw{Player: HumanSeat}` or `StateAwaitingDiscard{Player: HumanSeat}`
- **WHEN** the action footer renders
- **THEN** the rendered output contains `[FURITEN]` (Unicode renderer) or `(furiten)` (ASCII renderer)

#### Scenario: Furiten badge does not appear when human is not tenpai

- **GIVEN** the human's concealed hand has `Shanten >= 1`
- **AND** the current state is `StateAwaitingDiscard{Player: HumanSeat}`
- **WHEN** the action footer renders
- **THEN** the rendered output does NOT contain `[FURITEN]` or `(furiten)`

#### Scenario: Furiten badge does not appear when human is not in furiten

- **GIVEN** `Game.IsFuriten(HumanSeat)` is false
- **AND** the human's concealed hand has `Shanten == 0`
- **AND** the current state is `StateAwaitingDiscard{Player: HumanSeat}`
- **WHEN** the action footer renders
- **THEN** the rendered output does NOT contain `[FURITEN]` or `(furiten)`

#### Scenario: Furiten badge does not appear in the call window

- **GIVEN** the current state is `StateAwaitingClaims` with the human eligible to ron and in furiten
- **WHEN** the action footer renders
- **THEN** the rendered output uses the existing call-window `[R]on (furiten)` greyed Ron button (per the `Call Window Prompt` requirement) and does NOT add a separate `[FURITEN]` badge
