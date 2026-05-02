## ADDED Requirements

### Requirement: Bot Action Dispatch

The system SHALL dispatch bot actions through `handleBotTick` based on game state. The dispatcher SHALL invoke each of the following decisions in order, short-circuiting on the first that produces an action:

For `StateAwaitingDraw{Player: bot}`:
- Submit `InputDraw{}`.

For `StateAwaitingDiscard{Player: bot}`:
1. Tsumo check — build the bot's 14-tile `hand.Hand` with the drawn tile as `Winning` and call `calc.Analyze`. If non-nil, submit `InputDeclareTsumo` and stop.
2. Riichi check — invoke `Bot.ShouldRiichi(hand, scores[bot], wall.LiveRemaining(), isHandOpen)`. If `(declare=true, idx=<n>)`, submit `InputDiscard{Index: n, Riichi: true}` and stop.
3. Fallback — submit `InputDiscard{Index: bot.PickDiscard(hand)}`.

For `StateAwaitingClaims{Discarder: anySeat}`:
1. For each non-discarder seat in seat order East, South, West, North (skipping the discarder), evaluate in priority order: (a) ron — `calc.Analyze(concealed+discard) != nil` AND `!Game.IsFuriten(seat)` → `Claim{Kind: ClaimRon}`; (b) pon — `Bot.ShouldPon(hand, discard, isYakuhai, shanten)` → `Claim{Kind: ClaimPon}`; (c) chi (kamicha only) — `Bot.ShouldChi(hand, discard, discarder, seat)` returns a non-empty option list → `Claim{Kind: ClaimChi, ChiTiles: options[0]}`; (d) otherwise no claim.
2. Submit `InputResolveClaims{Claims: collectedMap}` in a single call. The engine's `ResolveClaims` enforces ron > pon > chi priority and the head-bump tiebreak.

When the human is also a non-discarder in claims state, the bot dispatcher SHALL NOT submit on the human's behalf — the human's own keypress drives their `Claim` (or pass via Space).

#### Scenario: Bot tsumo dispatched on winning draw

- **GIVEN** the active state is `AwaitingDiscard{Player: SeatEast}` and East's 14-tile hand wins yakufully
- **WHEN** the model receives `BotTickMsg`
- **THEN** the engine receives `InputDeclareTsumo` and transitions to `StateRoundOver{Outcome: OutcomeTsumo{Winner: SeatEast}}`

#### Scenario: Bot ron dispatched in claim window

- **GIVEN** the active state is `AwaitingClaims{Discard: 5p, Discarder: SeatEast}`
- **AND** SeatNorth's `concealed + 5p` forms a yaku-bearing winning hand and SeatNorth is not in furiten
- **WHEN** the model receives `BotTickMsg`
- **THEN** the engine receives `InputResolveClaims{Claims: {SeatNorth: ClaimRon}}` and transitions to `StateRoundOver{Outcome: OutcomeRon{Winner: SeatNorth, Loser: SeatEast}}`

#### Scenario: Bot pon dispatched in claim window

- **GIVEN** the active state is `AwaitingClaims{Discard: East-wind, Discarder: SeatEast}`
- **AND** SeatNorth has two East-wind tiles (yakuhai pon legal)
- **AND** SeatNorth is not at a winning shape on the discard
- **WHEN** the model receives `BotTickMsg`
- **THEN** the engine receives `InputResolveClaims{Claims: {SeatNorth: ClaimPon}}` and transitions to `StateAwaitingDiscard{Player: SeatNorth}`

#### Scenario: Bot riichi dispatched on tenpai discard turn

- **GIVEN** the active state is `AwaitingDiscard{Player: SeatWest}` and West's 14-tile hand has a discardable index leaving a tenpai 13-tile shape, score ≥1000, wall ≥4, hand concealed
- **AND** West's hand is NOT in a winning shape (no tsumo)
- **WHEN** the model receives `BotTickMsg`
- **THEN** the engine receives `InputDiscard{Index: <first-tenpai-leaving-index>, Riichi: true}`

#### Scenario: Multiple bots claim the same discard — resolver picks the winner

- **GIVEN** the active state is `AwaitingClaims{Discard: 5p, Discarder: SeatEast}`
- **AND** both SeatSouth (the human, hypothetically) and SeatWest can ron on 5p
- **WHEN** the bot dispatcher submits `InputResolveClaims` with both ron claims (in a hypothetical all-bot configuration)
- **THEN** the existing `ResolveClaims` head-bump rule selects the seat closest to the discarder going right around the table (SeatSouth in this example)

#### Scenario: Bot pass when no decision triggers

- **GIVEN** the active state is `AwaitingClaims` and no non-discarder seat has a legal claim
- **WHEN** the model receives `BotTickMsg`
- **THEN** the engine receives `InputResolveClaims{Claims: nil}` and the round advances to the next player's `AwaitingDraw`
