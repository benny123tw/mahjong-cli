# hand-calculator Specification

## Purpose

TBD - created by archiving change 'add-hand-calculator'. Update Purpose after archive.

## Requirements

### Requirement: Tile Notation Parsing

The system SHALL parse hand strings written in standard riichi notation, accepting tile codes `1m`–`9m` (man), `1p`–`9p` (pin), `1s`–`9s` (sou), `1z`–`7z` (honors: 1z=East, 2z=South, 3z=West, 4z=North, 5z=White dragon, 6z=Green dragon, 7z=Red dragon), and `0m`/`0p`/`0s` for red fives.

#### Scenario: Parse a 14-tile winning hand

- **WHEN** given the input string `1m2m3m4p5p6p7s8s9s1z1z2z2z2z`
- **THEN** the system returns a tile slice of length 14 with each tile decoded to its canonical tile ID, no tile flagged as red, and no parse error

##### Example: tile-code mapping

| Input | Tile ID | Suit  | Rank        | Red   |
| ----- | ------- | ----- | ----------- | ----- |
| 1m    | 0       | man   | 1           | false |
| 9m    | 8       | man   | 9           | false |
| 5p    | 13      | pin   | 5           | false |
| 0p    | 13      | pin   | 5           | true  |
| 1z    | 27      | honor | East wind   | false |
| 5z    | 31      | honor | White drag. | false |
| 7z    | 33      | honor | Red dragon  | false |

#### Scenario: Reject invalid tile codes

- **WHEN** the input contains `0z`, `8z`, `10m`, or any unrecognized character pair
- **THEN** the system returns a parse error naming the invalid token and its position

#### Scenario: Reject more than four copies of any tile

- **WHEN** the input contains a fifth copy of any single tile (with red fives counted toward the limit of their underlying tile value)
- **THEN** the system returns a validation error

#### Scenario: Reject hands of disallowed size

- **WHEN** the input has fewer than 13 or more than 14 concealed tiles, excluding any meld input
- **THEN** the system returns a validation error

---
### Requirement: Winning-Hand Detection

The system SHALL detect whether a 14-tile hand forms a winning shape, recognizing three forms: standard (four sets plus one pair), seven pairs (chiitoitsu), and thirteen orphans (kokushi musou).

#### Scenario: Standard form is detected

- **WHEN** given `1m2m3m4p5p6p7s8s9s1z1z2z2z2z`
- **THEN** the system reports a standard winning hand and returns at least one decomposition consisting of three sequences (1m2m3m, 4p5p6p, 7s8s9s), one triplet (2z2z2z), and one pair (1z1z)

#### Scenario: Chiitoitsu is detected

- **WHEN** given `1m1m4p4p7p7p2s2s5s5s8s8s1z1z`
- **THEN** the system reports chiitoitsu

#### Scenario: Kokushi musou is detected

- **WHEN** given `1m9m1p9p1s9s1z2z3z4z5z6z7z1m`
- **THEN** the system reports kokushi musou

#### Scenario: Chiitoitsu requires seven distinct pairs

- **WHEN** the hand contains four copies of the same tile arranged as two pairs of the same value
- **THEN** the system MUST NOT report chiitoitsu, and SHALL fall through to the standard-form check

#### Scenario: Non-winning hand

- **WHEN** given `1m2m3m4p5p6p7s8s9s1z1z2z2z3z`
- **THEN** the system reports that the hand is not a winning hand

---
### Requirement: Shanten and Machi Calculation

The system SHALL compute shanten (number of tile exchanges needed to reach tenpai) and the machi set (tiles that complete the hand) for any 13-tile concealed hand.

#### Scenario: Tenpai with tanki wait

- **WHEN** given the 13-tile hand `1m2m3m4m5m6m7p8p9p1s2s3s1z` (four sequences plus a single 1z)
- **THEN** shanten equals 0 and machi equals `{1z}`

#### Scenario: Non-tenpai hand reports shanten and empty machi

- **WHEN** given a hand that is not within one tile of a complete shape
- **THEN** shanten is at least 1 and machi is empty

##### Example: shanten/machi by shape

| 13-tile hand                   | Shanten | Machi    | Wait type |
| ------------------------------ | ------- | -------- | --------- |
| 1m2m3m4m5m6m7p8p9p1s2s3s1z     | 0       | 1z       | tanki     |
| 1m2m3m4p5p6p7s8s9s1z1z2z2z     | 0       | 1z, 2z   | shanpon   |
| 2m3m4p5p6p7p8p9p1s2s3s1z1z     | 0       | 1m, 4m   | ryanmen   |
| 1m2m3m4p5p6p7s7s8s8s9s9s1z     | 0       | 1z       | tanki     |
| 1m2m3m4p5p6p7s8s9s4z5z6z7z     | 2       | (empty)  | not tenpai |

---
### Requirement: Yaku Detection — V1 Set

The system SHALL detect the following yaku and report the han value of each, summed across all matched yaku:

| Yaku               | Han (concealed) | Han (open) | Conditions summary                                              |
| ------------------ | --------------- | ---------- | --------------------------------------------------------------- |
| Riichi             | 1               | n/a        | Player declared riichi; concealed only                          |
| Double riichi      | 2               | n/a        | Riichi declared on the player's first uninterrupted draw; concealed only; supersedes regular riichi |
| Ippatsu            | 1               | n/a        | Win on the player's next draw or any discard within the same go-around after declaring riichi, with no calls intervening; concealed only |
| Menzen tsumo       | 1               | n/a        | Win by tsumo with no called melds                               |
| Pinfu              | 1               | n/a        | All sequences, ryanmen wait, non-yakuhai pair, concealed        |
| Tanyao             | 1               | 1          | No terminals (1, 9) or honors                                   |
| Yakuhai            | 1 each          | 1 each     | Triplet/kan of round wind, seat wind, or any dragon             |
| Iipeikou           | 1               | n/a        | Two identical sequences in same suit, concealed only            |
| Haitei (raoyue)    | 1               | 1          | Win by tsumo on the very last drawn tile of the live wall       |
| Houtei (raoyui)    | 1               | 1          | Win by ron on the very last discard of the round                |
| Rinshan kaihou     | 1               | 1          | Win by tsumo on a tile drawn from the dead wall after declaring kan; SHALL NOT trigger in this change because kan is unsupported, but the detector SHALL exist and respond to the `Rinshan` flag |
| Chankan            | 1               | 1          | Win by ron on a tile that an opponent added to a previously-melded pon to form an open kan; SHALL NOT trigger in this change because kan is unsupported, but the detector SHALL exist and respond to the `Chankan` flag |
| Tenhou             | yakuman         | n/a        | Dealer wins by tsumo on their initial 14-tile deal with no intervening calls; concealed only |
| Chiihou            | yakuman         | n/a        | Non-dealer wins by tsumo on their first uninterrupted draw with no intervening calls; concealed only |
| Toitoi             | 2               | 2          | All triplets/kans                                               |
| Honitsu            | 3               | 2          | One numeric suit plus honors                                    |
| Sanshoku doujun    | 2               | 1          | Same numeric sequence in all three suits                        |
| Ittsuu             | 2               | 1          | 1-2-3, 4-5-6, 7-8-9 all in the same suit                        |
| Chanta             | 2               | 1          | Every set and the pair contains at least one terminal-or-honor  |
| Junchan            | 3               | 2          | Every set and the pair contains a terminal AND no honor anywhere|
| Honroutou          | 2               | 2          | Every tile is a terminal or honor (no simples 2–8)              |
| Sanankou           | 2               | 2          | Three concealed triplets at agari (winning triplet via shanpon-ron is treated as open) |
| Sanshoku doukou    | 2               | 2          | Same numeric triplet in all three suits at the same rank        |
| Shousangen         | 2 + yakuhai     | 2 + yakuhai| Two dragon triplets plus a dragon pair (yakuhai still counts each dragon triplet) |
| Ryanpeikou         | 3               | n/a        | Two distinct iipeikou shapes, concealed only; supersedes iipeikou|
| Chinitsu           | 6               | 5          | One numeric suit only, no honors                                |

The eight rows above the existing detectors (Double riichi, Ippatsu, Haitei, Houtei, Rinshan kaihou, Chankan, Tenhou, Chiihou) are the **Group C situational yaku**. Their detection depends on game-loop state surfaced via eight new bool flags on `yaku.Context`: `DoubleRiichi`, `Ippatsu`, `Haitei`, `Houtei`, `Rinshan`, `Chankan`, `Tenhou`, `Chiihou`. Each detector SHALL match if and only if its corresponding flag is true at agari (subject to its concealment requirement where applicable).

#### Scenario: Multiple yaku stack

- **GIVEN** context with riichi declared, win by tsumo, seat East, round East
- **WHEN** given the winning hand `3m4m5m6m7m2p3p4p5p6p7p3s3s2m` (winning tile = the last tile, 2m, completing 2m3m4m as a ryanmen)
- **THEN** the system reports `[riichi, menzen tsumo, pinfu, tanyao]` totalling 4 han

#### Scenario: Yakuhai requires applicable wind or dragon

- **GIVEN** seat East, round East
- **WHEN** the winning hand contains a triplet of East wind
- **THEN** yakuhai contributes 2 han (round wind and seat wind both apply)

#### Scenario: Non-applicable wind triplet contributes no yakuhai

- **GIVEN** seat South, round East
- **WHEN** the winning hand contains a triplet of West wind
- **THEN** yakuhai contributes 0 han for that triplet

#### Scenario: Iipeikou disallowed when hand is open

- **WHEN** the hand contains an iipeikou shape but the player has called pon, chi, or open kan
- **THEN** iipeikou is not counted

#### Scenario: Open hand downgrades honitsu and sanshoku and ittsuu

- **WHEN** the winning hand qualifies for honitsu, sanshoku doujun, or ittsuu and contains at least one called meld
- **THEN** honitsu reports 2 han, sanshoku doujun reports 1 han, ittsuu reports 1 han

#### Scenario: Pinfu requires all four shape conditions

- **WHEN** any of these fail — all four sets are sequences; the wait is a ryanmen; the pair is not yakuhai; the hand is concealed
- **THEN** pinfu is not counted

#### Scenario: Chinitsu requires single numeric suit and no honors

- **WHEN** the winning hand contains only tiles of one numeric suit (m / p / s) and no honor tiles
- **THEN** chinitsu matches at 6 han concealed, 5 han open
- **AND** honitsu does not also match (honitsu requires at least one honor)

#### Scenario: Sanankou counts only triplets concealed at agari

- **WHEN** the winning decomposition contains at least three triplets that were all concealed at the moment of agari
- **THEN** sanankou matches at 2 han

##### Example: ron on a shanpon completion suppresses sanankou

- **GIVEN** a winning hand `1m1m1m4m4m4m7m7m7m9m9m5m5m5m` decomposed as four triplets (1m, 4m, 7m, 5m) and a pair (9m), where the winning tile 5m completed the 5m triplet via shanpon-ron
- **WHEN** the player wins by ron
- **THEN** the 5m triplet is treated as open for the sanankou count, leaving only three concealed triplets (1m, 4m, 7m) — sanankou matches at 2 han
- **WHEN** the player instead wins by tsumo on the same hand
- **THEN** all four triplets count as concealed; sanankou still matches but the hand also satisfies suuankou (deferred yakuman, not detected in this change)

#### Scenario: Sanshoku doukou requires the same numeric triplet across all three suits

- **WHEN** the winning decomposition contains triplets of the same rank in man, pin, and sou
- **THEN** sanshoku doukou matches at 2 han, in both concealed and open hands

##### Example: 4m / 4p / 4s triplets

- **GIVEN** winning hand `4m4m4m4p4p4p4s4s4s2m3m4m5p5p` (triplets at rank 4 in all three suits, plus 2m3m4m sequence and 5p pair)
- **THEN** sanshoku doukou matches at 2 han

#### Scenario: Chanta requires every set and the pair to contain a terminal or honor

- **WHEN** every meld in the winning decomposition (the four sets and the pair) contains at least one terminal (1 or 9) or honor tile
- **THEN** chanta matches at 2 han concealed, 1 han open

#### Scenario: Junchan is chanta with no honors anywhere

- **WHEN** every meld contains at least one terminal AND no honor tile appears anywhere in the hand
- **THEN** junchan matches at 3 han concealed, 2 han open
- **AND** chanta does not separately match (junchan supersedes chanta when honors are absent)

#### Scenario: Honroutou requires only terminals and honors

- **WHEN** every tile in the winning hand is a terminal (1 or 9 of a numeric suit) or an honor
- **THEN** honroutou matches at 2 han, in both concealed and open hands
- **AND** the hand is necessarily either toitoi (no sequences possible) or chiitoitsu

#### Scenario: Shousangen requires two dragon triplets plus a dragon pair

- **WHEN** the winning decomposition contains triplets of two different dragons and a pair of the third dragon
- **THEN** shousangen matches at 2 han, in addition to the two yakuhai han contributed by the two dragon triplets

#### Scenario: Ryanpeikou supersedes iipeikou

- **WHEN** the winning decomposition has FormStandard with two distinct iipeikou shapes (two pairs of identical sequences), concealed
- **THEN** ryanpeikou matches at 3 han, AND iipeikou does not separately match for that decomposition
- **WHEN** the hand is open
- **THEN** ryanpeikou does not match (concealed only)

#### Scenario: Ippatsu matches when flag is set and hand is concealed

- **GIVEN** `Context{Riichi: true, Ippatsu: true}` and a concealed winning hand
- **WHEN** the system evaluates yaku
- **THEN** ippatsu matches at 1 han alongside riichi
- **GIVEN** the same flags but the hand is open (impossible in practice because riichi requires concealed, but tested defensively)
- **THEN** ippatsu SHALL NOT match

#### Scenario: Double riichi suppresses regular riichi

- **GIVEN** `Context{Riichi: true, DoubleRiichi: true}` and a concealed winning hand
- **WHEN** the system evaluates yaku
- **THEN** double riichi matches at 2 han AND regular riichi SHALL NOT separately match

#### Scenario: Haitei matches a tsumo on the last live-wall draw

- **GIVEN** `Context{IsTsumo: true, Haitei: true}` and a winning hand
- **WHEN** the system evaluates yaku
- **THEN** haitei matches at 1 han
- **GIVEN** `Context{IsTsumo: false, Haitei: true}` (ron with the haitei flag — invalid combination)
- **THEN** haitei SHALL NOT match

#### Scenario: Houtei matches a ron on the very last discard

- **GIVEN** `Context{IsTsumo: false, Houtei: true}` and a winning hand
- **WHEN** the system evaluates yaku
- **THEN** houtei matches at 1 han
- **GIVEN** `Context{IsTsumo: true, Houtei: true}` (tsumo with the houtei flag — invalid combination)
- **THEN** houtei SHALL NOT match

#### Scenario: Tenhou matches dealer's first-deal tsumo

- **GIVEN** `Context{Tenhou: true, IsTsumo: true, Seat: East}` and a concealed winning hand
- **WHEN** the system evaluates yaku
- **THEN** tenhou matches as a yakuman
- **GIVEN** the same context with a non-dealer seat or an open hand
- **THEN** tenhou SHALL NOT match (the game loop SHALL NOT set the flag in those cases)

#### Scenario: Chiihou matches non-dealer's first-draw tsumo

- **GIVEN** `Context{Chiihou: true, IsTsumo: true, Seat: South}` and a concealed winning hand with no calls anywhere in the round before this draw
- **WHEN** the system evaluates yaku
- **THEN** chiihou matches as a yakuman
- **GIVEN** the same context but the dealer or another player called pon/chi/kan before this draw
- **THEN** chiihou SHALL NOT match (the game loop SHALL clear the flag on the first call)

#### Scenario: Rinshan and Chankan detectors stay dormant in this change

- **GIVEN** `Context{Rinshan: true}` or `Context{Chankan: true}`
- **WHEN** the system evaluates yaku
- **THEN** the detector code path exists and would match at 1 han
- **AND** in this change the game loop never sets these flags because kan is unsupported, so neither yaku is observable in actual play; the detectors SHALL nonetheless be unit-tested with the flags forced on so the future kan-support change wires them in without engine changes

---
### Requirement: Fu Calculation

The system SHALL compute fu for any winning hand. Standard-form fu SHALL be rounded up to the nearest 10. Pinfu-tsumo SHALL produce a flat 20 fu with no rounding. Chiitoitsu SHALL produce a flat 25 fu with no rounding.

##### Example: fu components

| Component                                    | Fu  |
| -------------------------------------------- | --- |
| Base                                         | 20  |
| Menzen ron bonus                             | +10 |
| Tsumo bonus (not applied to pinfu-tsumo)     | +2  |
| Concealed triplet of simples (2–8)           | +4  |
| Open triplet of simples                      | +2  |
| Concealed triplet of terminals or honors     | +8  |
| Open triplet of terminals or honors          | +4  |
| Concealed kan of simples                     | +16 |
| Open kan of simples                          | +8  |
| Concealed kan of terminals or honors         | +32 |
| Open kan of terminals or honors              | +16 |
| Tanki, kanchan, or penchan wait              | +2  |
| Yakuhai pair (round/seat wind, any dragon)   | +2  |
| Open hand with otherwise zero-fu shape       | +2 (kuipinfu) |

#### Scenario: Pinfu tsumo flat 20

- **GIVEN** a winning hand qualifying for pinfu, won by tsumo
- **THEN** fu equals 20 with no rounding applied

#### Scenario: Chiitoitsu flat 25

- **GIVEN** a winning chiitoitsu hand
- **THEN** fu equals 25 with no rounding applied

#### Scenario: Standard fu rounds up to nearest 10

- **GIVEN** computed fu of 32
- **THEN** reported fu equals 40

#### Scenario: Concealed terminal triplets contribute 8 each

- **GIVEN** a winning hand with concealed triplets of 1z and 2z, won by ron, with menzen
- **THEN** fu equals `20 + 10 (menzen ron) + 8 (1z triplet) + 8 (2z triplet) = 46 → 50`

---
### Requirement: Final Score Calculation

The system SHALL convert (han, fu, dealer flag, win type) to a final point award using the standard riichi score table and SHALL apply mangan, haneman, baiman, sanbaiman, and kazoe-yakuman caps.

##### Example: non-dealer ron payout boundaries

| Han | Fu  | Score | Tier             |
| --- | --- | ----- | ---------------- |
| 1   | 30  | 1000  | normal           |
| 3   | 30  | 3900  | normal           |
| 4   | 30  | 7700  | normal           |
| 5   | any | 8000  | mangan           |
| 6   | any | 12000 | haneman          |
| 8   | any | 16000 | baiman           |
| 11  | any | 24000 | sanbaiman        |
| 13  | any | 32000 | kazoe yakuman    |

#### Scenario: Dealer scores 1.5x non-dealer

- **GIVEN** dealer ron, 3 han 30 fu
- **THEN** score equals 5800 (compared to 3900 for non-dealer with the same han/fu)

#### Scenario: True yakuman caps non-yakuman han

- **WHEN** the hand contains a true yakuman (e.g., kokushi musou) plus other yaku
- **THEN** reported tier is yakuman (32000 non-dealer, 48000 dealer) and other yaku are listed for display but contribute no han to the final score

---
### Requirement: Decomposition Selection

The system SHALL, when a winning hand admits multiple valid decompositions, select the decomposition whose final point award is highest. When two decompositions produce identical point awards, the system SHALL pick deterministically by lexicographic ordering of the decomposition's canonical string form.

#### Scenario: Higher-scoring decomposition wins

- **GIVEN** a winning hand and game context where two valid decompositions yield different yaku totals
- **WHEN** the engine evaluates both
- **THEN** the engine reports the decomposition with higher final points as the chosen one

#### Scenario: Deterministic tie-break

- **GIVEN** two decompositions producing identical han, fu, and final points
- **WHEN** the engine selects between them
- **THEN** the chosen decomposition is the one whose canonical string form sorts first lexicographically

---
### Requirement: CLI Command Surface

The system SHALL expose hand-calculator functionality through `mahjong calc <hand-string>` and SHALL accept the following flags: `--seat E|S|W|N` (default S, non-dealer), `--round E|S` (default E), `--riichi` (boolean), `--tsumo` (boolean; if absent, win is treated as ron), `--dora <tile>` (repeatable), `--uradora <tile>` (repeatable; ignored unless `--riichi`).

#### Scenario: Successful winning-hand analysis

- **WHEN** the user runs `mahjong calc 1m2m3m4p5p6p7s8s9s1z1z2z2z2z --tsumo`
- **THEN** the process exits with code 0 and stdout contains a shanten line, a yaku list with per-yaku han, a fu breakdown line, and a final-points line

##### Example: stdout for an invocation chosen so the hand has exactly one yaku

- **WHEN** user runs `mahjong calc 1m2m3m4p5p6p7s8s9s1z1z2z2z2z --tsumo --seat W` (West seat means 2z is not yakuhai)
- **THEN** stdout contains lines equivalent to:
  - `Shanten: -1 (winning)`
  - `Yaku: Menzen tsumo (1)`
  - `Fu: 40`
  - `Han: 1  Fu: 40  Points: 1500 (non-dealer tsumo: 400/700)`

#### Scenario: Tenpai-only analysis for 13-tile input

- **WHEN** the user runs `mahjong calc <thirteen-tile-hand>` with no winning shape
- **THEN** stdout reports shanten and machi but no yaku, fu, or points

#### Scenario: Invalid hand string fails fast

- **WHEN** the user runs `mahjong calc <unparseable>`
- **THEN** the process exits with non-zero code and stderr contains a parse error identifying the offending token
