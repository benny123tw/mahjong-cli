## MODIFIED Requirements

### Requirement: Furiten Query

The system SHALL expose `Game.IsFuriten(seat Seat) bool` returning true when EITHER permanent furiten OR temporary furiten is active for the seat. Permanent furiten holds when any tile in the seat's own discard pond matches a tile ID in the seat's current machi (computed via `hand.Machi` on the seat's concealed hand at exactly 13 tiles). Temporary furiten holds when an opponent has discarded a tile that completes a winning shape on the seat's hand (`hand.IsWinning(concealed + discard) == true`) since the seat's last own draw, AND the seat did not submit a ron claim for that discard. Temporary furiten clears when the seat takes their next own draw (`stepFromAwaitingDraw` resets the per-seat flag). When the seat's hand is not exactly 13 tiles, `IsFuriten` SHALL return false (the machi is undefined for non-tenpai shapes, and yaku-less winning shapes still trigger the temporary lockout but require a 13-tile concealed hand to be meaningful).

#### Scenario: Permanent furiten when machi tile is in own pond

- **GIVEN** the seat's 13-tile hand has machi `{4m, 7m}` and the seat's discard pond contains `4m`
- **WHEN** `Game.IsFuriten(seat)` is called
- **THEN** the result is `true`

#### Scenario: Not furiten when machi tiles are absent and no winning tile passed

- **GIVEN** the seat's 13-tile hand has machi `{4m, 7m}`, the seat's pond contains `1z, 9m, 5p`, and no opponent has discarded `4m` or `7m` since the seat's last draw
- **WHEN** `Game.IsFuriten(seat)` is called
- **THEN** the result is `false`

#### Scenario: Temporary furiten when opponent discards machi tile and seat passes

- **GIVEN** the seat's 13-tile hand wins on `5p` and the seat is NOT in permanent furiten
- **AND** an opponent discards `5p` and the seat does not submit a ron claim
- **WHEN** `Game.IsFuriten(seat)` is called before the seat's next draw
- **THEN** the result is `true` (temporary furiten armed)

#### Scenario: Temporary furiten clears on the seat's next own draw

- **GIVEN** the seat is in temporary furiten (machi tile passed since last draw)
- **WHEN** the seat takes their next own draw via `stepFromAwaitingDraw`
- **THEN** `Game.IsFuriten(seat)` returns false (temporary flag reset; permanent furiten still applies if relevant)

#### Scenario: Furiten query on non-tenpai hand returns false

- **GIVEN** the seat's 13-tile hand has shanten ≥1 (machi is empty)
- **WHEN** `Game.IsFuriten(seat)` is called
- **THEN** the result is `false`

---

### Requirement: Bot Decision Strategy

Bot opponents SHALL play a single hand-coded strategy with the following rules:

| Decision | Rule |
| -------- | ---- |
| Discard | When NO opponent has declared riichi, pick the tile maximizing isolation (no neighbor within 2 ranks in same suit, no copies elsewhere). Honors and terminals score highest. Tiebreak: lowest tile ID. When AT LEAST ONE opponent has declared riichi, pick the tile maximizing a danger-aware score = isolation - 2000 × danger, where danger is 0 for genbutsu (tile-ID matches any tile in a riichi-declarer's pond), 1 for suji-safe against a riichi-declarer (rank pair table — 1↔4, 7↔4, 2↔5, 8↔5, 3↔6, 9↔6 — same suit), and 2 otherwise. The 2000× constant guarantees any safe tile is preferred over any unsafe tile regardless of isolation difference. When multiple riichi declarers exist, danger is the MIN across them (the safest declaration determines the score). |
| Pon (yakuhai) | Always when bot has 2 copies of a discarded yakuhai tile (round wind, seat wind, or any dragon) |
| Pon (non-yakuhai) | 50% probability when bot has 2 copies AND bot is at shanten ≤ 2 |
| Chi | 40% probability, only from kamicha, only when discard completes a 2-tile partial (ryanmen / kanchan / penchan) in bot's hand |
| Kan | Never |
| Riichi | When the bot's 14-tile hand has a discardable index that leaves a tenpai 13-tile shape AND the bot is concealed (no called melds) AND the bot's score is ≥1000 AND `Wall.LiveRemaining()` is ≥4. The bot SHALL pick the FIRST scanned index (0..len-1) whose post-discard hand has shanten=0 and submit `InputDiscard{Index: idx, Riichi: true}`. |
| Ron | When `calc.Analyze` on the bot's `concealed + discard` returns a non-nil result AND `Game.IsFuriten(seat)` returns false (permanent OR temporary furiten blocks ron) |
| Tsumo | When `calc.Analyze` on the bot's 14-tile hand (after drawing) returns a non-nil result |

All probabilistic decisions SHALL use a PRNG seeded from the same seed as the wall, so games reproduce deterministically. Bot riichi tile-choice and danger-aware discard tile-choice are deterministic and SHALL NOT consume from the PRNG.

#### Scenario: Bot prefers genbutsu against riichi declarer

- **GIVEN** the human declared riichi and their pond contains `5p`
- **AND** a bot's hand contains both `5p` (genbutsu) and `9m` (unknown danger)
- **AND** the bot's isolation score for `9m` is higher than for `5p` in the absence of danger
- **WHEN** the bot's `AwaitingDiscard` state is dispatched
- **THEN** the bot discards `5p` (the genbutsu) — the danger penalty (-2000 × 2 = -4000) on `9m` outweighs the isolation difference

#### Scenario: Bot prefers suji over unknown when no genbutsu available

- **GIVEN** the human declared riichi and their pond contains `4p` (no other relevant tiles)
- **AND** a bot's hand contains `1p` (suji per the 1↔4 rule), `7p` (suji per the 7↔4 rule), and `3m` (unknown)
- **AND** none of these tiles are genbutsu (`4p` is in pond but the bot doesn't hold `4p`)
- **WHEN** the bot's `AwaitingDiscard` state is dispatched
- **THEN** the bot discards `1p` or `7p` (both danger 1) over `3m` (danger 2), provided isolation scores don't extreme-favor `3m`. The danger gap (-2000 vs -4000) dominates a typical isolation gap

#### Scenario: Bot ron blocked by temporary furiten

- **GIVEN** an opponent discarded a bot's machi tile T on a previous turn AND the bot did not ron on that discard
- **AND** before the bot's next own draw, a different opponent discards T again
- **WHEN** the bot's claim window evaluates ron
- **THEN** the bot does NOT submit ron — `Game.IsFuriten(bot)` returns true via the temporary-furiten arm and the dispatcher passes

#### Scenario: Bot does not chi from non-kamicha

- **GIVEN** a bot has 4m + 5m and the discarded 6m
- **WHEN** the discarder is not the bot's kamicha
- **THEN** the bot does not call chi regardless of dice roll

#### Scenario: Bot tsumo on a yaku-bearing draw

- **GIVEN** a bot draws a tile completing a yaku-bearing winning shape (14-tile hand has `calc.Analyze` non-nil)
- **WHEN** the bot's `AwaitingDiscard` state is dispatched
- **THEN** the bot submits `InputDeclareTsumo` and the round advances to `StateRoundOver{Outcome: OutcomeTsumo{...}}`

#### Scenario: Bot declares riichi when tenpai-after-discard

- **GIVEN** a bot's 14-tile hand has at least one discardable index that leaves a 13-tile shape with shanten=0
- **AND** the bot is concealed, score ≥1000, wall remaining ≥4
- **WHEN** the bot's `AwaitingDiscard` state is dispatched
- **THEN** the bot submits `InputDiscard{Index: <first-tenpai-leaving-index>, Riichi: true}`
