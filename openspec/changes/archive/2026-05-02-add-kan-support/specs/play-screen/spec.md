## MODIFIED Requirements

### Requirement: Keybinding Map

The system SHALL bind the documented keymap. Cursor-movement keys SHALL update the focused tile within the player's hand. Action keys SHALL drive real game-state transitions: `D` or Enter discards the focused tile; `R` declares riichi (when legal — concealed hand at tenpai with at least 1000 points); `T` declares tsumo on a winning drawn tile; `P` / `C` / `K` / `R` / `Space` operate inside the call-window prompt. The `K` key in `StateAwaitingDiscard{Player: HumanSeat}` SHALL open the kan picker (ankan or shouminkan) when the human has any eligible kan declaration; in `StateAwaitingClaims` it SHALL submit a `ClaimKan` minkan claim when the human's hand contains 3 of the discarded tile. The `K` key SHALL be greyed when the human has declared riichi (riichi-and-kan interaction deferred).

#### Scenario: Cursor moves right with arrow or l

- **WHEN** the cursor is at hand position i (0-indexed) and the player presses `→` or `l`
- **THEN** the cursor moves to position min(i+1, hand_length-1)

#### Scenario: Cursor moves left with arrow or h

- **WHEN** the cursor is at hand position i and the player presses `←` or `h`
- **THEN** the cursor moves to position max(i-1, 0)

#### Scenario: Number key jumps cursor

- **WHEN** the player presses a key in the range `1`–`9`
- **THEN** the cursor moves to the nth tile (1-indexed) of the hand if n ≤ hand_length, otherwise the cursor moves to the last tile

#### Scenario: Discard advances game state

- **GIVEN** the underlying state is `AwaitingDiscard{Player: human}` and the cursor is on tile T
- **WHEN** the human presses `D` or Enter
- **THEN** tile T is removed from the human's hand and appended to the human's discard zone, and state advances to `AwaitingClaims{Discard: T, Discarder: human}`

#### Scenario: Riichi declaration is rejected when illegal

- **WHEN** the human presses `R` while their hand is open or not yet at tenpai
- **THEN** state does not change and a brief footer feedback indicates riichi is illegal in the current state

#### Scenario: Kan key opens picker when ankan is eligible

- **GIVEN** the underlying state is `AwaitingDiscard{Player: HumanSeat}` and the human's concealed hand contains four `5p` tiles
- **WHEN** the human presses `K`
- **THEN** the footer shows a kan picker listing `5p` as the eligible declaration
- **AND** pressing the corresponding selector key submits `InputDeclareAnkan{TileID: tile.P5}`

#### Scenario: Kan key submits minkan in claim window

- **GIVEN** the underlying state is `AwaitingClaims{Discard: 5p, Discarder: SeatEast}` and the human has three `5p` in hand
- **WHEN** the human presses `K`
- **THEN** the engine receives `Claim{Kind: ClaimKan}` for the human seat and the round transitions to `StateAwaitingDiscard{Player: HumanSeat}` after the kan resolves

#### Scenario: Kan key greyed during human's riichi

- **GIVEN** the human has declared riichi (`Game.IsRiichiDeclared(HumanSeat)` is true)
- **WHEN** the human presses `K` in `StateAwaitingDiscard`
- **THEN** the kan declaration does not fire and the footer shows a greyed indicator

##### Example: full keymap

| Key            | Behavior in this change                                                                  |
| -------------- | ---------------------------------------------------------------------------------------- |
| `←`, `→`, h, l | Move cursor across the player's hand                                                     |
| 1–9            | Jump cursor to nth tile (also picks kan option when picker is open)                      |
| D, Enter       | Discard tile under cursor (when in `AwaitingDiscard` state)                              |
| R              | Declare riichi (when legal: concealed, tenpai, ≥ 1000 points)                            |
| T              | Tsumo on the drawn tile (when winning hand with at least one yaku, including rinshan)    |
| P              | Pon (in call window only, when legal)                                                    |
| C              | Chi (in call window only, only from kamicha, when legal)                                 |
| K              | Open ankan/shouminkan picker (own turn) or submit minkan claim (call window); greyed when in riichi |
| R              | Ron (in call window OR chankan window when winning hand with at least one yaku)          |
| Space          | Pass in call/chankan window; no-op outside call windows                                  |
| ?              | Machi/yaku peek (cached `hand.Shanten` + `hand.Machi` lookup)                            |
| q, Ctrl+C      | Quit cleanly                                                                             |
