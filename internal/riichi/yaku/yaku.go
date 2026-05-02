// Package yaku detects v1 yaku given a winning Decomposition, Hand, and game
// Context. Each detector is independent and returns zero or more Matches; the
// orchestrator (Evaluate) runs every detector against a single decomposition
// and applies one cross-cutting rule — if any yakuman is matched, non-yakuman
// matches are dropped.
package yaku

import (
	"slices"

	"github.com/benny123tw/mahjong-cli/internal/riichi/hand"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

type Context struct {
	SeatWind  uint8
	RoundWind uint8
	Riichi    bool

	// Group C situational flags. The game loop populates these at win-check
	// time. Detectors read each flag directly; the game loop is responsible
	// for the timing rules (e.g., clearing Ippatsu on any intervening call).
	//
	// Rinshan and Chankan ship in v1 but are never set to true because kan
	// is unsupported. Their detectors exist so add-kan-support wires them in
	// without engine changes.
	Ippatsu      bool
	Haitei       bool
	Houtei       bool
	Rinshan      bool
	Chankan      bool
	DoubleRiichi bool
	Tenhou       bool
	Chiihou      bool
}

type Match struct {
	Name      string
	Han       int
	IsYakuman bool
}

type Detector func(d hand.Decomposition, h hand.Hand, ctx Context) []Match

func Detectors() []Detector {
	return []Detector{
		detectKokushi,
		detectTenhou,
		detectChiihou,
		detectRiichi,
		detectDoubleRiichi,
		detectIppatsu,
		detectMenzenTsumo,
		detectPinfu,
		detectTanyao,
		detectYakuhai,
		detectIipeikou,
		detectToitoi,
		detectHonitsu,
		detectSanshokuDoujun,
		detectIttsuu,
		detectChinitsu,
		detectHonroutou,
		detectChanta,
		detectJunchan,
		detectSanankou,
		detectSanshokuDoukou,
		detectShousangen,
		detectRyanpeikou,
		detectHaitei,
		detectHoutei,
		detectRinshan,
		detectChankan,
		detectSankantsu,
		detectSuukantsu,
		detectSuuankou,
		detectDaisangen,
		detectDaisuushii,
		detectShousuushii,
		detectTsuuiisou,
		detectChinroutou,
		detectRyuuiisou,
		detectChuurenPoutou,
	}
}

// --- Group D: kan-aware yaku.

// kanCount returns the number of kan-kind CalledMelds (ankan, minkan, or
// shouminkan) on the winning hand. Used by sankantsu/suukantsu detectors.
func kanCount(h hand.Hand) int {
	n := 0
	for _, cm := range h.CalledMelds {
		switch cm.Kind {
		case hand.CalledAnkan, hand.CalledMinkan, hand.CalledShouminkan:
			n++
		}
	}
	return n
}

func detectSankantsu(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if kanCount(h) != 3 {
		return nil
	}
	return []Match{{Name: "Sankantsu", Han: 2}}
}

func detectSuukantsu(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if kanCount(h) != 4 {
		return nil
	}
	return []Match{{Name: "Suukantsu", Han: 13, IsYakuman: true}}
}

// suuankouTripletConcealed determines whether the triplet at base ID `B`
// counts as concealed for suuankou. A called pon/minkan/shouminkan at B
// makes it open; an ankan at B keeps it concealed; no called meld at B
// means the triplet was formed from concealed tiles. Additionally, when
// the win is by ron on a tile equal to B AND the wait is shanpon (pair
// base != B), the triplet is downgraded to open per riichi convention.
func suuankouTripletConcealed(b uint8, h hand.Hand, pairBase uint8) bool {
	for _, cm := range h.CalledMelds {
		if cm.BaseID != b {
			continue
		}
		switch cm.Kind {
		case hand.CalledPon, hand.CalledMinkan, hand.CalledShouminkan:
			return false
		}
	}
	if !h.IsTsumo && h.Winning.ID == b && b != pairBase {
		return false
	}
	return true
}

func detectSuuankou(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form != hand.FormStandard {
		return nil
	}
	pairBase := d.Pair().Base
	concealedTriplets := 0
	for _, m := range d.Sets() {
		if m.Kind != hand.MeldTriplet {
			return nil
		}
		if suuankouTripletConcealed(m.Base, h, pairBase) {
			concealedTriplets++
		}
	}
	if concealedTriplets != 4 {
		return nil
	}
	return []Match{{Name: "Suuankou", Han: 13, IsYakuman: true}}
}

// Evaluate runs every detector and returns the matched yaku for one
// decomposition. Two cross-cutting rules are applied after detection:
//   - If any yakuman matches, non-yakuman yaku are dropped.
//   - Ryanpeikou supersedes Iipeikou (when both match the same decomposition,
//     iipeikou is dropped).
func Evaluate(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	var all []Match
	for _, det := range Detectors() {
		all = append(all, det(d, h, ctx)...)
	}

	hasRyanpeikou := false
	for _, m := range all {
		if m.Name == "Ryanpeikou" {
			hasRyanpeikou = true
			break
		}
	}
	if hasRyanpeikou {
		filtered := all[:0]
		for _, m := range all {
			if m.Name == "Iipeikou" {
				continue
			}
			filtered = append(filtered, m)
		}
		all = filtered
	}

	// Double riichi supersedes Riichi: when both match the same evaluation,
	// drop the regular riichi line.
	hasDoubleRiichi := false
	for _, m := range all {
		if m.Name == "Double riichi" {
			hasDoubleRiichi = true
			break
		}
	}
	if hasDoubleRiichi {
		filtered := all[:0]
		for _, m := range all {
			if m.Name == "Riichi" {
				continue
			}
			filtered = append(filtered, m)
		}
		all = filtered
	}

	// Suuankou supersedes Sanankou: when both match the same evaluation,
	// drop the sanankou line. The yakuman filter below would also drop
	// sanankou (suuankou is yakuman, sanankou is not), but the explicit
	// rule mirrors the ryanpeikou/double-riichi supersession blocks and
	// stays correct if the yakuman filter is later relaxed.
	hasSuuankou := false
	for _, m := range all {
		if m.Name == "Suuankou" {
			hasSuuankou = true
			break
		}
	}
	if hasSuuankou {
		filtered := all[:0]
		for _, m := range all {
			if m.Name == "Sanankou" {
				continue
			}
			filtered = append(filtered, m)
		}
		all = filtered
	}

	// Daisuushii supersedes Shousuushii: when both match the same evaluation,
	// drop the shousuushii line. Same defensive pattern as suuankou-sanankou.
	hasDaisuushii := false
	for _, m := range all {
		if m.Name == "Daisuushii" {
			hasDaisuushii = true
			break
		}
	}
	if hasDaisuushii {
		filtered := all[:0]
		for _, m := range all {
			if m.Name == "Shousuushii" {
				continue
			}
			filtered = append(filtered, m)
		}
		all = filtered
	}

	hasYakuman := false
	for _, m := range all {
		if m.IsYakuman {
			hasYakuman = true
			break
		}
	}
	if !hasYakuman {
		return all
	}
	kept := all[:0]
	for _, m := range all {
		if m.IsYakuman {
			kept = append(kept, m)
		}
	}
	return kept
}

func detectKokushi(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form == hand.FormKokushi {
		return []Match{{Name: "Kokushi musou", Han: 13, IsYakuman: true}}
	}
	return nil
}

func detectRiichi(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if ctx.Riichi && !h.Open {
		return []Match{{Name: "Riichi", Han: 1}}
	}
	return nil
}

func detectMenzenTsumo(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if h.IsTsumo && !h.Open {
		return []Match{{Name: "Menzen tsumo", Han: 1}}
	}
	return nil
}

func detectPinfu(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form != hand.FormStandard || h.Open {
		return nil
	}
	for _, m := range d.Sets() {
		if m.Kind != hand.MeldSequence {
			return nil
		}
	}
	if isYakuhaiTile(d.Pair().Base, ctx) {
		return nil
	}
	if !ryanmenPossible(d, h.Winning) {
		return nil
	}
	return []Match{{Name: "Pinfu", Han: 1}}
}

func detectTanyao(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	for _, t := range h.Concealed {
		if t.IsTerminalOrHonor() {
			return nil
		}
	}
	return []Match{{Name: "Tanyao", Han: 1}}
}

func detectYakuhai(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form != hand.FormStandard {
		return nil
	}
	var matches []Match
	for _, m := range d.Sets() {
		if m.Kind != hand.MeldTriplet {
			continue
		}
		t := tile.Tile{ID: m.Base}
		if !t.IsHonor() {
			continue
		}
		if m.Base == ctx.RoundWind {
			matches = append(matches, Match{Name: "Yakuhai (round wind)", Han: 1})
		}
		if m.Base == ctx.SeatWind {
			matches = append(matches, Match{Name: "Yakuhai (seat wind)", Han: 1})
		}
		if t.IsDragon() {
			matches = append(matches, Match{Name: "Yakuhai (" + dragonName(m.Base) + ")", Han: 1})
		}
	}
	return matches
}

func detectIipeikou(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form != hand.FormStandard || h.Open {
		return nil
	}
	sets := d.Sets()
	for i := range sets {
		for j := i + 1; j < len(sets); j++ {
			if sets[i].Kind == hand.MeldSequence && sets[j].Kind == hand.MeldSequence &&
				sets[i].Base == sets[j].Base {
				return []Match{{Name: "Iipeikou", Han: 1}}
			}
		}
	}
	return nil
}

func detectToitoi(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form != hand.FormStandard {
		return nil
	}
	for _, m := range d.Sets() {
		if m.Kind != hand.MeldTriplet {
			return nil
		}
	}
	return []Match{{Name: "Toitoi", Han: 2}}
}

func detectHonitsu(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form == hand.FormKokushi {
		return nil
	}
	var suit tile.Suit
	suitSet := false
	hasHonor := false
	for _, t := range h.Concealed {
		if t.IsHonor() {
			hasHonor = true
			continue
		}
		if !suitSet {
			suit = t.Suit()
			suitSet = true
		} else if t.Suit() != suit {
			return nil
		}
	}
	if !suitSet || !hasHonor {
		return nil
	}
	han := 3
	if h.Open {
		han = 2
	}
	return []Match{{Name: "Honitsu", Han: han}}
}

func detectSanshokuDoujun(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form != hand.FormStandard {
		return nil
	}
	for rank := uint8(1); rank <= 7; rank++ {
		hasMan := false
		hasPin := false
		hasSou := false
		for _, m := range d.Sets() {
			if m.Kind != hand.MeldSequence {
				continue
			}
			switch m.Base {
			case tile.M1 + rank - 1:
				hasMan = true
			case tile.P1 + rank - 1:
				hasPin = true
			case tile.S1 + rank - 1:
				hasSou = true
			}
		}
		if hasMan && hasPin && hasSou {
			han := 2
			if h.Open {
				han = 1
			}
			return []Match{{Name: "Sanshoku doujun", Han: han}}
		}
	}
	return nil
}

func detectIttsuu(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form != hand.FormStandard {
		return nil
	}
	for _, base := range []uint8{tile.M1, tile.P1, tile.S1} {
		has123, has456, has789 := false, false, false
		for _, m := range d.Sets() {
			if m.Kind != hand.MeldSequence {
				continue
			}
			switch m.Base {
			case base:
				has123 = true
			case base + 3:
				has456 = true
			case base + 6:
				has789 = true
			}
		}
		if has123 && has456 && has789 {
			han := 2
			if h.Open {
				han = 1
			}
			return []Match{{Name: "Ittsuu", Han: han}}
		}
	}
	return nil
}

func isYakuhaiTile(id uint8, ctx Context) bool {
	t := tile.Tile{ID: id}
	if t.IsDragon() {
		return true
	}
	if t.IsWind() && (id == ctx.SeatWind || id == ctx.RoundWind) {
		return true
	}
	return false
}

func dragonName(id uint8) string {
	switch id {
	case tile.Haku:
		return "Haku"
	case tile.Hatsu:
		return "Hatsu"
	case tile.Chun:
		return "Chun"
	}
	return "?"
}

// ryanmenPossible reports whether at least one sequence in the decomposition
// can be interpreted as the winning sequence with a ryanmen completion (the
// winning tile is at one end of the sequence and the opposite end isn't
// against the suit's edge — a 7-8-9 with winning=7 or a 1-2-3 with winning=3
// is penchan, never ryanmen; a sequence with winning at the middle is
// kanchan).
func ryanmenPossible(d hand.Decomposition, winning tile.Tile) bool {
	for _, m := range d.Sets() {
		if m.Kind != hand.MeldSequence {
			continue
		}
		if !m.Contains(winning.ID) {
			continue
		}
		baseRank := tile.Tile{ID: m.Base}.Rank()
		pos := winning.ID - m.Base
		switch pos {
		case 0:
			if baseRank != 7 {
				return true
			}
		case 1:
			// kanchan
		case 2:
			if baseRank != 1 {
				return true
			}
		}
	}
	return false
}

// --- Group A: composition-based detectors (chinitsu, honroutou, chanta, junchan).

func detectChinitsu(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form == hand.FormKokushi {
		return nil
	}
	var suit tile.Suit
	suitSet := false
	for _, t := range h.Concealed {
		if t.IsHonor() {
			return nil
		}
		if !suitSet {
			suit = t.Suit()
			suitSet = true
			continue
		}
		if t.Suit() != suit {
			return nil
		}
	}
	if !suitSet {
		return nil
	}
	han := 6
	if h.Open {
		han = 5
	}
	return []Match{{Name: "Chinitsu", Han: han}}
}

func detectHonroutou(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form == hand.FormKokushi {
		return nil
	}
	for _, t := range h.Concealed {
		if !t.IsTerminalOrHonor() {
			return nil
		}
	}
	return []Match{{Name: "Honroutou", Han: 2}}
}

func detectChanta(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form == hand.FormKokushi {
		return nil
	}
	if !handContainsHonor(h) {
		return nil
	}
	if !everyMeldHasYaochuhai(d) {
		return nil
	}
	if d.Form == hand.FormChiitoitsu {
		return []Match{{Name: "Chanta", Han: 2}}
	}
	han := 2
	if h.Open {
		han = 1
	}
	return []Match{{Name: "Chanta", Han: han}}
}

func detectJunchan(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form == hand.FormKokushi {
		return nil
	}
	if handContainsHonor(h) {
		return nil
	}
	if !everyMeldHasTerminal(d) {
		return nil
	}
	if d.Form == hand.FormChiitoitsu {
		return []Match{{Name: "Junchan", Han: 3}}
	}
	han := 3
	if h.Open {
		han = 2
	}
	return []Match{{Name: "Junchan", Han: han}}
}

// --- Group A: meld-shape detectors (sanankou, sanshoku doukou, shousangen, ryanpeikou).

func detectSanankou(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form != hand.FormStandard {
		return nil
	}
	concealedTriplets := 0
	for _, m := range d.Sets() {
		if m.Kind != hand.MeldTriplet {
			continue
		}
		if h.Open {
			continue
		}
		// Ron on a shanpon completion: the winning tile is in this triplet AND
		// the win was ron (not tsumo). That triplet is treated as open for
		// sanankou-counting purposes.
		if !h.IsTsumo && m.Base == h.Winning.ID {
			continue
		}
		concealedTriplets++
	}
	if concealedTriplets < 3 {
		return nil
	}
	return []Match{{Name: "Sanankou", Han: 2}}
}

func detectSanshokuDoukou(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form != hand.FormStandard {
		return nil
	}
	for rank := uint8(1); rank <= 9; rank++ {
		hasMan := false
		hasPin := false
		hasSou := false
		for _, m := range d.Sets() {
			if m.Kind != hand.MeldTriplet {
				continue
			}
			switch m.Base {
			case tile.M1 + rank - 1:
				hasMan = true
			case tile.P1 + rank - 1:
				hasPin = true
			case tile.S1 + rank - 1:
				hasSou = true
			}
		}
		if hasMan && hasPin && hasSou {
			return []Match{{Name: "Sanshoku doukou", Han: 2}}
		}
	}
	return nil
}

func detectShousangen(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form != hand.FormStandard {
		return nil
	}
	pairBase := d.Pair().Base
	if !(tile.Tile{ID: pairBase}).IsDragon() {
		return nil
	}
	dragonTriplets := 0
	for _, m := range d.Sets() {
		if m.Kind == hand.MeldTriplet && (tile.Tile{ID: m.Base}).IsDragon() {
			dragonTriplets++
		}
	}
	if dragonTriplets != 2 {
		return nil
	}
	return []Match{{Name: "Shousangen", Han: 2}}
}

func detectRyanpeikou(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form != hand.FormStandard || h.Open {
		return nil
	}
	sets := d.Sets()
	for _, m := range sets {
		if m.Kind != hand.MeldSequence {
			return nil
		}
	}
	// Count sequence-base occurrences. Ryanpeikou requires exactly two distinct
	// bases each appearing twice (two iipeikou shapes).
	counts := map[uint8]int{}
	for _, m := range sets {
		counts[m.Base]++
	}
	if len(counts) != 2 {
		return nil
	}
	for _, c := range counts {
		if c != 2 {
			return nil
		}
	}
	return []Match{{Name: "Ryanpeikou", Han: 3}}
}

// --- Shared helpers for Group A.

func handContainsHonor(h hand.Hand) bool {
	for _, t := range h.Concealed {
		if t.IsHonor() {
			return true
		}
	}
	return false
}

func everyMeldHasYaochuhai(d hand.Decomposition) bool {
	for _, m := range d.Melds {
		if !meldHasYaochuhai(m) {
			return false
		}
	}
	return true
}

func everyMeldHasTerminal(d hand.Decomposition) bool {
	for _, m := range d.Melds {
		if !meldHasTerminal(m) {
			return false
		}
	}
	return true
}

func meldHasYaochuhai(m hand.Meld) bool {
	return slices.ContainsFunc(m.Tiles(), hand.IsYaochuhai)
}

func meldHasTerminal(m hand.Meld) bool {
	for _, id := range m.Tiles() {
		if (tile.Tile{ID: id}).IsTerminal() {
			return true
		}
	}
	return false
}

// --- Group C: situational / turn-aware detectors. The game loop populates
// the corresponding bool flag on Context; the detectors are one-line
// flag reads plus the obvious gating (concealment, win type, dealer).

func detectIppatsu(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if !ctx.Ippatsu || h.Open {
		return nil
	}
	return []Match{{Name: "Ippatsu", Han: 1}}
}

func detectDoubleRiichi(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if !ctx.DoubleRiichi || h.Open {
		return nil
	}
	return []Match{{Name: "Double riichi", Han: 2}}
}

func detectHaitei(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if !ctx.Haitei || !h.IsTsumo {
		return nil
	}
	return []Match{{Name: "Haitei raoyue", Han: 1}}
}

func detectHoutei(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if !ctx.Houtei || h.IsTsumo {
		return nil
	}
	return []Match{{Name: "Houtei raoyui", Han: 1}}
}

func detectRinshan(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if !ctx.Rinshan || !h.IsTsumo {
		return nil
	}
	return []Match{{Name: "Rinshan kaihou", Han: 1}}
}

func detectChankan(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if !ctx.Chankan || h.IsTsumo {
		return nil
	}
	return []Match{{Name: "Chankan", Han: 1}}
}

func detectTenhou(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if !ctx.Tenhou || h.Open || !h.IsTsumo {
		return nil
	}
	if ctx.SeatWind != tile.EastWind {
		return nil
	}
	return []Match{{Name: "Tenhou", Han: 13, IsYakuman: true}}
}

func detectChiihou(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if !ctx.Chiihou || h.Open || !h.IsTsumo {
		return nil
	}
	if ctx.SeatWind == tile.EastWind {
		return nil
	}
	return []Match{{Name: "Chiihou", Han: 13, IsYakuman: true}}
}

// --- Group B: non-kan yakuman.

func detectDaisangen(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form != hand.FormStandard {
		return nil
	}
	hasHaku, hasHatsu, hasChun := false, false, false
	for _, m := range d.Sets() {
		if m.Kind != hand.MeldTriplet {
			continue
		}
		switch m.Base {
		case tile.Haku:
			hasHaku = true
		case tile.Hatsu:
			hasHatsu = true
		case tile.Chun:
			hasChun = true
		}
	}
	if hasHaku && hasHatsu && hasChun {
		return []Match{{Name: "Daisangen", Han: 13, IsYakuman: true}}
	}
	return nil
}

func detectDaisuushii(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form != hand.FormStandard {
		return nil
	}
	winds := [4]bool{}
	for _, m := range d.Sets() {
		if m.Kind != hand.MeldTriplet {
			continue
		}
		switch m.Base {
		case tile.EastWind:
			winds[0] = true
		case tile.SouthWind:
			winds[1] = true
		case tile.WestWind:
			winds[2] = true
		case tile.NorthWind:
			winds[3] = true
		}
	}
	if winds[0] && winds[1] && winds[2] && winds[3] {
		return []Match{{Name: "Daisuushii", Han: 13, IsYakuman: true}}
	}
	return nil
}

func detectShousuushii(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form != hand.FormStandard {
		return nil
	}
	pairBase := d.Pair().Base
	pairTile := tile.Tile{ID: pairBase}
	if !pairTile.IsWind() {
		return nil
	}
	windTriplets := 0
	for _, m := range d.Sets() {
		if m.Kind != hand.MeldTriplet {
			continue
		}
		if (tile.Tile{ID: m.Base}).IsWind() {
			windTriplets++
		}
	}
	if windTriplets != 3 {
		return nil
	}
	return []Match{{Name: "Shousuushii", Han: 13, IsYakuman: true}}
}

func detectTsuuiisou(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form == hand.FormKokushi {
		return nil
	}
	for _, t := range h.Concealed {
		if !t.IsHonor() {
			return nil
		}
	}
	return []Match{{Name: "Tsuuiisou", Han: 13, IsYakuman: true}}
}

func detectChinroutou(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form != hand.FormStandard {
		return nil
	}
	for _, t := range h.Concealed {
		if !t.IsTerminal() || t.IsHonor() {
			return nil
		}
	}
	return []Match{{Name: "Chinroutou", Han: 13, IsYakuman: true}}
}

// greenTileIDs is the set of tiles that count for ryuuiisou: 2s/3s/4s/6s/8s
// plus Hatsu (the green dragon). Using a uint64 bitmask keyed on tile ID
// keeps the hot-path lookup branch-free.
var greenTileIDs = func() (mask uint64) {
	for _, id := range []uint8{tile.S2, tile.S3, tile.S4, tile.S6, tile.S8, tile.Hatsu} {
		mask |= 1 << id
	}
	return mask
}()

func detectRyuuiisou(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form == hand.FormKokushi {
		return nil
	}
	for _, t := range h.Concealed {
		if greenTileIDs&(1<<t.ID) == 0 {
			return nil
		}
	}
	return []Match{{Name: "Ryuuiisou", Han: 13, IsYakuman: true}}
}

func detectChuurenPoutou(d hand.Decomposition, h hand.Hand, ctx Context) []Match {
	if d.Form != hand.FormStandard {
		return nil
	}
	if h.Open || len(h.CalledMelds) > 0 {
		return nil
	}
	if len(h.Concealed) != 14 {
		return nil
	}
	// All tiles must share a single numeric suit. Determine the suit from
	// the first tile and bail on any honor or off-suit tile.
	if h.Concealed[0].IsHonor() {
		return nil
	}
	suit := h.Concealed[0].Suit()
	for _, t := range h.Concealed[1:] {
		if t.IsHonor() || t.Suit() != suit {
			return nil
		}
	}
	// Per-rank counts in the suit (ranks 1..9).
	var counts [10]int
	for _, t := range h.Concealed {
		counts[t.Rank()]++
	}
	// Subtract one occurrence of the winning tile's rank to get the
	// remaining 13 tiles. They must form 1-1-1-2-3-4-5-6-7-8-9-9-9.
	if h.Winning.Suit() != suit {
		return nil
	}
	counts[h.Winning.Rank()]--
	want := [10]int{0, 3, 1, 1, 1, 1, 1, 1, 1, 3}
	if counts != want {
		return nil
	}
	return []Match{{Name: "Chuuren poutou", Han: 13, IsYakuman: true}}
}
