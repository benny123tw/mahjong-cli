## MODIFIED Requirements

### Requirement: Keybinding Map

The system SHALL bind the documented keymap. Cursor-movement keys SHALL update the focused tile within the player's hand. Action keys SHALL drive real game-state transitions: `D` or Enter discards the focused tile; `R` declares riichi (when legal — concealed hand at tenpai with at least 1000 points) on the player's own turn, and ron in the call window; `T` declares tsumo on a winning drawn tile; `P` / `C` / `K` / `Space` operate inside the call-window prompt.

The `K` key SHALL declare a kan according to the current state:

- In `StateAwaitingDiscard{Player: HumanSeat}`, `K` SHALL declare an ankan when the human's concealed hand contains a 4-of-a-kind, OR a shouminkan when the human has an open `MeldPon` plus the matching 4th tile in their concealed hand. When both kinds of declaration are eligible, the engine prioritizes ankan first, then shouminkan, and within each kind picks the lowest tile-ID match deterministically (no popup picker — the player's only handle is the cursor and the K key, and the deterministic priority avoids requiring a multi-option picker UI for v1).
- In `StateAwaitingClaims`, `K` SHALL submit `Claim{Kind: ClaimKan}` for the human seat when the human's hand contains 3 of the discarded tile (minkan).
- When the human has declared riichi, `K` SHALL be inert in `StateAwaitingDiscard` — the engine rejects kans during riichi for v1, and the play layer surfaces the rejection via `ackText`.

The action footer SHALL render the `K` key's live/greyed style based on real-time state:

- During `StateAwaitingDiscard{Player: HumanSeat}`, `K` SHALL be live when ankan or shouminkan is eligible (the legality predicate composes 4-of-a-kind detection with open-pon-plus-matching-tile detection); greyed when neither holds; and greyed unconditionally when the human has declared riichi.
- During `StateAwaitingClaims`, the call-window footer (rendered by `RenderCallFooter`) already surfaces `[K]an` live when `CanKan(humanHand, discard)` is true — that path is unchanged.
- Outside both states, `K` SHALL be greyed.

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

#### Scenario: K key declares ankan when 4-of-a-kind is concealed

- **GIVEN** the underlying state is `AwaitingDiscard{Player: HumanSeat}`, the human's hand contains four `5p` tiles, and the human has not declared riichi
- **WHEN** the human presses `K`
- **THEN** the engine receives `InputDeclareAnkan{TileID: tile.P5}`
- **AND** the underlying state is no longer `AwaitingDiscard{Player: HumanSeat}` immediately after (the engine has begun the rinshan replacement-draw flow)
- **AND** the model's `ackText` reads `ankan declared`

#### Scenario: K key declares shouminkan when held tile matches an open pon

- **GIVEN** the underlying state is `AwaitingDiscard{Player: HumanSeat}`, the human has an open `MeldPon{Tiles: [1m, 1m, 1m]}`, the human's concealed hand contains a `1m`, and the human has not declared riichi
- **WHEN** the human presses `K`
- **THEN** the engine receives `InputDeclareShouminkan{TileID: tile.M1}`
- **AND** the underlying state transitions to `StateAwaitingChankan{Declarer: HumanSeat, UpgradeTile: 1m}`
- **AND** the model's `ackText` reads `shouminkan declared`

#### Scenario: K key submits minkan in claim window

- **GIVEN** the underlying state is `AwaitingClaims{Discard: 5p, Discarder: SeatEast}` and the human has three `5p` in hand
- **WHEN** the human presses `K`
- **THEN** the engine receives `Claim{Kind: ClaimKan}` for the human seat and the round transitions to `StateAwaitingDiscard{Player: HumanSeat}` after the kan resolves

#### Scenario: K key greyed during human's riichi

- **GIVEN** the human has declared riichi (`Game.IsRiichiDeclared(HumanSeat)` is true)
- **WHEN** the human presses `K` in `StateAwaitingDiscard`
- **THEN** the kan declaration does not fire and the model's `ackText` reads `kan: not allowed during riichi`

#### Scenario: Action footer K is live when any kan is eligible

- **GIVEN** the underlying state is `AwaitingDiscard{Player: HumanSeat}` and the human's hand contains four `5p` tiles (ankan eligible)
- **WHEN** the View renders the action footer
- **THEN** the rendered output styles `K Kan` with `liveKeyStyle` rather than `greyedKeyStyle`

##### Example: K liveness across states and hand contents

| State                                  | Hand condition                                  | Riichi | Footer K style |
| -------------------------------------- | ----------------------------------------------- | ------ | -------------- |
| AwaitingDiscard{Human}                 | 4× 1m in concealed                              | no     | live           |
| AwaitingDiscard{Human}                 | 1m in concealed + open MeldPon{1m,1m,1m}        | no     | live           |
| AwaitingDiscard{Human}                 | 4× 1m in concealed                              | yes    | greyed         |
| AwaitingDiscard{Human}                 | only 3× 1m in concealed, no matching open pon   | no     | greyed         |
| AwaitingDraw{Human}                    | any                                             | any    | greyed         |
