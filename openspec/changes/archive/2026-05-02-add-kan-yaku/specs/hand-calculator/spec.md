## MODIFIED Requirements

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
| Rinshan kaihou     | 1               | 1          | Win by tsumo on a tile drawn from the dead wall after declaring kan |
| Chankan            | 1               | 1          | Win by ron on a tile that an opponent added to a previously-melded pon to form an open kan |
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
| Sankantsu          | 2               | 2          | Exactly three kan melds in the winning decomposition (any kan kind: ankan, minkan, shouminkan) |
| Suukantsu          | yakuman         | yakuman    | Exactly four kan melds in the winning decomposition by one player (any kan kind); Sankantsu does NOT also match |
| Suuankou           | yakuman         | n/a        | Four concealed triplets at agari; ankan counts as concealed; ron on a shanpon-completed triplet downgrades that triplet to open and disables suuankou; ron-on-tanki and tsumo preserve suuankou |

The eight rows above the existing detectors (Double riichi, Ippatsu, Haitei, Houtei, Rinshan kaihou, Chankan, Tenhou, Chiihou) are the **Group C situational yaku**. Their detection depends on game-loop state surfaced via eight new bool flags on `yaku.Context`: `DoubleRiichi`, `Ippatsu`, `Haitei`, `Houtei`, `Rinshan`, `Chankan`, `Tenhou`, `Chiihou`. Each detector SHALL match if and only if its corresponding flag is true at agari (subject to its concealment requirement where applicable).

The three new kan-related rows (Sankantsu, Suukantsu, Suuankou) form the **Group D kan-aware yaku**. Their detection requires `add-kan-support` to be in place: kans must be representable in the winning decomposition as `MeldKan` entries with a `KanKind` field distinguishing ankan from minkan/shouminkan. Suuankou treats ankan as a concealed triplet for its count.

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
- **THEN** all four triplets count as concealed; sanankou still matches AND suuankou also matches as a yakuman (suuankou supersedes sanankou)

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
- **THEN** ryanpeikou does not match

#### Scenario: Sankantsu matches when the winning decomposition contains exactly three kans

- **WHEN** the winning decomposition contains exactly three `MeldKan` entries (any combination of `KanAnkan`, `KanMinkan`, or `KanShouminkan`)
- **THEN** sankantsu matches at 2 han, in both concealed and open hands
- **AND** suukantsu does NOT match (suukantsu requires four kans)

##### Example: three ankans plus an open hand still matches sankantsu

- **GIVEN** a player declares ankan on 1m, ankan on 9m, and shouminkan on 5p (after an earlier pon of 5p), then wins by tsumo on a completing tile
- **WHEN** the winning decomposition is inspected
- **THEN** the three `MeldKan` entries are counted regardless of `KanKind`, and sankantsu reports 2 han

#### Scenario: Suukantsu matches when the winning decomposition contains exactly four kans

- **WHEN** the winning decomposition contains exactly four `MeldKan` entries by the same player (any combination of kan kinds), reducing the concealed hand to exactly the pair (two tiles)
- **THEN** suukantsu matches as a yakuman
- **AND** sankantsu does NOT also match (suukantsu supersedes sankantsu)
- **AND** the rinshan replacement that completes the fourth kan is the most common path to this hand; if the win is by rinshan kaihou, both yaku stack as separate matches

#### Scenario: Suuankou requires four concealed triplets

- **WHEN** the winning decomposition contains four triplet-shaped melds, each of which is either a concealed (uncalled) triplet or a `MeldKan` with `KanKind == KanAnkan`
- **AND** the win is by tsumo, OR the win is by ron with a tanki (single-tile pair) wait
- **THEN** suuankou matches as a yakuman
- **AND** the suuankou match supersedes any sanankou or toitoi match for the same decomposition (suuankou subsumes them as a yakuman; the lower yaku are not separately reported)

##### Example: four ankans tsumo

- **GIVEN** a player declares ankan on 1m, ankan on 4m, ankan on 7m, and ankan on 9m, then wins by tsumo with the pair already complete
- **WHEN** the winning decomposition is inspected
- **THEN** the four `MeldKan{KanKind: KanAnkan}` entries each count as concealed triplets, and suuankou matches as a yakuman

#### Scenario: Ron-on-shanpon disables suuankou

- **GIVEN** a winning hand `1m1m1m4m4m4m7m7m7m9m9m5m5m5m` decomposed as four triplets (1m, 4m, 7m, 5m) and a pair (9m), where the winning tile 5m completed the 5m triplet via shanpon-ron
- **WHEN** the player wins by ron
- **THEN** the 5m triplet is treated as open for the suuankou check, leaving only three concealed triplets, and suuankou does NOT match
- **AND** sanankou still matches at 2 han for the three remaining concealed triplets

#### Scenario: Ron-on-tanki preserves suuankou

- **GIVEN** a winning hand `1m1m1m4m4m4m7m7m7m9m9m9m5m5m` decomposed as four triplets (1m, 4m, 7m, 9m) and a pair (5m), where the winning tile 5m completed the pair via tanki-ron
- **WHEN** the player wins by ron
- **THEN** all four triplets remain concealed (the wait did not complete a triplet), and suuankou matches as a yakuman

##### Example: ankan-plus-three-ankou hand

| Triplet 1 | Triplet 2 | Triplet 3 | Triplet 4 (form)         | Wait shape | Win type | Suuankou |
| --------- | --------- | --------- | ------------------------ | ---------- | -------- | -------- |
| 1m ankan  | 4m ankou  | 7m ankou  | 9m ankou (concealed)     | tanki on 5m| tsumo    | matches  |
| 1m ankan  | 4m ankou  | 7m ankou  | 5m ankou (shanpon-ron)   | shanpon    | ron      | no match |
| 1m ankan  | 4m ankou  | 7m ankou  | 9m ankou (tanki-ron 5m)  | tanki      | ron      | matches  |
| 1m minkan | 4m ankou  | 7m ankou  | 9m ankou                 | tanki      | tsumo    | no match (1m kan is open) |
