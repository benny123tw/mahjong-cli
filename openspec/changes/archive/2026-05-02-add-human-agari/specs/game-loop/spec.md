## ADDED Requirements

### Requirement: Riichi Declaration

The system SHALL accept `InputDiscard{Riichi: true}` from a seat in `StateAwaitingDiscard` only when ALL of the following preconditions hold: the seat has zero called melds (concealed hand), the seat's score is at least 1000 points, the live wall has at least 4 tiles remaining, and the seat is in tenpai after the proposed discard. When all preconditions hold, the system SHALL deduct 1000 points from the seat as a riichi deposit, mark the seat as riichi-declared, open an ippatsu window for the seat, and complete the discard transition normally (advance to `StateAwaitingClaims`). When any precondition fails, the system SHALL return `ErrIllegalRiichi` and leave game state unchanged.

#### Scenario: Riichi declared on a tenpai concealed hand with funds

- **GIVEN** the human is in `StateAwaitingDiscard{Player: Human}` with a 14-tile concealed hand whose post-discard form is tenpai
- **AND** the human has 25000 points and the live wall has 60 tiles remaining
- **WHEN** the system receives `InputDiscard{Index: 13, Riichi: true}`
- **THEN** the seat's score becomes 24000 (1000-point deposit deducted)
- **AND** the seat is marked riichi-declared
- **AND** state advances to `StateAwaitingClaims{Discarder: Human}`

#### Scenario: Riichi rejected when hand is open

- **GIVEN** the human has previously called pon (one open meld) and is in `StateAwaitingDiscard{Player: Human}`
- **WHEN** the system receives `InputDiscard{Index: 13, Riichi: true}`
- **THEN** the system returns `ErrIllegalRiichi`
- **AND** game state is unchanged (same hand, same state, no point deduction)

#### Scenario: Riichi rejected when wall has fewer than 4 tiles

- **GIVEN** the live wall has exactly 3 tiles remaining and the human is in `StateAwaitingDiscard{Player: Human}` with a tenpai post-discard hand
- **WHEN** the system receives `InputDiscard{Index: 13, Riichi: true}`
- **THEN** the system returns `ErrIllegalRiichi`
- **AND** game state is unchanged

#### Scenario: Riichi rejected when post-discard hand is not tenpai

- **GIVEN** the human is in `StateAwaitingDiscard{Player: Human}` and `hand.Shanten` of the post-discard 13-tile hand is ≥1
- **WHEN** the system receives `InputDiscard{Index: 5, Riichi: true}`
- **THEN** the system returns `ErrIllegalRiichi`
- **AND** game state is unchanged

#### Scenario: Riichi rejected when seat has fewer than 1000 points

- **GIVEN** the human's score is 800 (e.g., after prior noten payments)
- **AND** the human is in `StateAwaitingDiscard{Player: Human}` on a tenpai post-discard hand
- **WHEN** the system receives `InputDiscard{Index: 13, Riichi: true}`
- **THEN** the system returns `ErrIllegalRiichi`

##### Example: deposit deduction at declaration

- **GIVEN** human score = 25000, wall remaining = 60, hand = `[1m,2m,3m,4m,5m,6m,7m,8m,9m,1p,1p,2p,2p, drawn=3p]` (chiitoitsu-no, but the post-discard hand `[1m..9m, 1p, 1p, 2p, 2p]` is shanten=1 — pinfu wait... actually let's pick a valid tenpai)
- **WHEN** the human discards `9m` with `Riichi: true` from a hand that lands at tenpai
- **THEN** post-state: score=24000, riichiDeclared[Human]=true, ippatsuLive[Human]=true

---

### Requirement: Riichi-Restricted Discard

After a seat has declared riichi, the system SHALL only accept `InputDiscard` from that seat when the discard index points to the just-drawn tile (the rightmost tile in the seat's hand, at index `len(hand)-1`). Any `InputDiscard` with a different index from a riichi-declared seat SHALL return `ErrIllegalDiscard` and leave game state unchanged. This restriction applies starting on the seat's NEXT turn after the riichi-declaring discard (the declaring discard itself is selected freely; subsequent discards are forced).

#### Scenario: Post-riichi discard locked to drawn tile

- **GIVEN** the human declared riichi on a previous turn and is now in `StateAwaitingDiscard{Player: Human}` with a 14-tile hand
- **WHEN** the system receives `InputDiscard{Index: 5}` (a sorted-hand tile, not the drawn tile)
- **THEN** the system returns `ErrIllegalDiscard`

#### Scenario: Post-riichi discard accepted at index 13

- **GIVEN** the human declared riichi on a previous turn and is now in `StateAwaitingDiscard{Player: Human}` with a 14-tile hand
- **WHEN** the system receives `InputDiscard{Index: 13}` (the just-drawn tile)
- **THEN** the discard transition completes normally (advance to `StateAwaitingClaims`)

---

### Requirement: Ippatsu Window Tracking

The system SHALL track an ippatsu window for each seat that has declared riichi. The window opens at the moment the seat's riichi-declaring discard transitions to `StateAwaitingClaims`. The window closes when EITHER (a) the seat makes their next non-riichi discard since declaration (i.e., they drew and chose not to win), OR (b) any seat (including the declarer) executes a successful pon or chi call. While the window is open, the seat may either ron on any opponent's discard or tsumo on their own next draw and earn ippatsu. When the seat wins while the window is open, the system SHALL pass `Ippatsu = true` to `calc.Analyze` via `Game.contextForWin`; otherwise `Ippatsu = false`.

#### Scenario: Ippatsu when riichi → opponents pass → win on own next draw

- **GIVEN** the human declares riichi and the next state is `StateAwaitingClaims{Discarder: Human}` with no claimants
- **AND** West / North / East all draw and discard with no calls intervening
- **WHEN** the human draws their next tile and declares tsumo
- **THEN** `calc.Context{Ippatsu: true}` is passed to `calc.Analyze`

#### Scenario: Ippatsu broken by intervening pon

- **GIVEN** the human declares riichi
- **AND** before the human's next draw, West calls pon on East's discard
- **WHEN** the human eventually wins (tsumo or ron)
- **THEN** `calc.Context{Ippatsu: false}` is passed to `calc.Analyze`

#### Scenario: Ippatsu closes on the seat's next non-tsumo discard

- **GIVEN** the human declares riichi on turn 5 and the ippatsu window is open
- **AND** the round proceeds with no calls; the human's draw on turn 6 happens but they do not tsumo
- **AND** the human discards the drawn tile (forced by riichi-restricted-discard)
- **WHEN** the human eventually wins on a later turn
- **THEN** `calc.Context{Ippatsu: false}` is passed to `calc.Analyze` (window closed at turn 6 own-discard)

---

### Requirement: Double Riichi Detection

When `InputDiscard{Riichi: true}` succeeds AND the declaring seat has not yet discarded any tile this round AND no calls have happened this round, the system SHALL mark the declaration as a "double riichi". When the seat subsequently wins, `calc.Context{DoubleRiichi: true}` SHALL be passed to `calc.Analyze`. The standard `Riichi` flag SHALL remain set as well; the calc layer dedupes (existing yaku-detection contract).

#### Scenario: Double riichi on first uninterrupted intake

- **GIVEN** the dealer (East) draws their first tile and the round has zero prior discards and no prior calls
- **AND** the dealer's post-discard hand is tenpai
- **WHEN** the dealer submits `InputDiscard{Index: 13, Riichi: true}`
- **THEN** the seat's `doubleRiichi[East] = true` is recorded
- **AND** when the dealer subsequently wins, `calc.Context.DoubleRiichi = true` is passed

#### Scenario: Riichi declared after any discard is regular riichi only

- **GIVEN** East draws and discards (regular discard, no riichi) on turn 1
- **AND** South draws on turn 2 and decides to declare riichi
- **WHEN** South submits `InputDiscard{Index: 13, Riichi: true}`
- **THEN** `doubleRiichi[South] = false` and `riichiDeclared[South] = true`
- **AND** when South wins, `calc.Context.Riichi = true, DoubleRiichi = false`

---

### Requirement: Furiten Query

The system SHALL expose `Game.IsFuriten(seat Seat) bool` returning true when ANY tile in the seat's own discard pond matches ANY tile ID in the seat's current machi (computed via `hand.Machi` on the seat's concealed hand at exactly 13 tiles). When the seat's hand is not exactly 13 tiles, `IsFuriten` SHALL return false (the machi is undefined for non-tenpai shapes). When the seat is in tenpai with no machi tiles in own pond, `IsFuriten` returns false. Permanent furiten only — temporary furiten across multiple opponent discards is out of scope for v1.

#### Scenario: Furiten when machi tile is in own pond

- **GIVEN** the human's 13-tile hand has machi `{4m, 7m}` and the human's discard pond contains `4m`
- **WHEN** `Game.IsFuriten(Human)` is called
- **THEN** the result is `true`

#### Scenario: Not furiten when machi tiles are absent from own pond

- **GIVEN** the human's 13-tile hand has machi `{4m, 7m}` and the human's discard pond contains `1z, 9m, 5p`
- **WHEN** `Game.IsFuriten(Human)` is called
- **THEN** the result is `false`

#### Scenario: Furiten query on non-tenpai hand returns false

- **GIVEN** the human's 13-tile hand has shanten ≥1 (machi is empty)
- **WHEN** `Game.IsFuriten(Human)` is called
- **THEN** the result is `false`

---

### Requirement: Human Ron From Claim Window

The system SHALL accept `InputResolveClaims{Claims: {HumanSeat: Claim{Kind: ClaimRon}}}` in `StateAwaitingClaims` when ALL of the following hold: `calc.Analyze` on the human's `concealed + discard` returns a non-nil result, AND `Game.IsFuriten(HumanSeat)` returns false. The transition SHALL go through the existing `stepFromAwaitingClaims` ron path: build the winning `hand.Hand`, call `calc.Analyze` with the populated `contextForWin`, and transition to `StateRoundOver{Outcome: OutcomeRon{...}}`. When `calc.Analyze` returns nil (no yaku) OR `IsFuriten` is true, the system SHALL return `ErrYakulessWin` (existing sentinel — re-used for the no-yaku case) or a new sentinel `ErrFuritenRon` respectively, and leave game state unchanged.

#### Scenario: Human ron on a yaku-bearing discard

- **GIVEN** the human is in `StateAwaitingClaims{Discard: 5p, Discarder: East}`
- **AND** the human's `concealed + 5p` forms a yaku-bearing winning shape
- **AND** `Game.IsFuriten(Human)` returns false
- **WHEN** the system receives `InputResolveClaims{Claims: {Human: ClaimRon}}`
- **THEN** state advances to `StateRoundOver{Outcome: OutcomeRon{Winner: Human, Loser: East, ...}}`

#### Scenario: Human ron rejected when furiten

- **GIVEN** the human is in `StateAwaitingClaims{Discard: 5p, Discarder: East}`
- **AND** the human's hand would win on 5p but the human's own pond contains 5p
- **WHEN** the system receives `InputResolveClaims{Claims: {Human: ClaimRon}}`
- **THEN** the system returns `ErrFuritenRon`
- **AND** game state is unchanged
