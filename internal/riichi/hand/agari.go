package hand

import "github.com/benny123tw/mahjong-cli/internal/riichi/tile"

// Decompose returns every valid winning decomposition of the concealed
// portion of h. It checks the three agari forms — standard (4 sets + 1 pair),
// chiitoitsu (7 pairs), kokushi musou (13 unique yaochuhai with one pair) —
// and accumulates results from all three.
//
// Decompose operates only on h.Concealed; melds are not modeled in v1.
// For tenpai analysis use Shanten / Machi (see machi.go), not Decompose.
func Decompose(h Hand) []Decomposition {
	if len(h.Concealed) != 14 {
		return nil
	}
	counts := h.Counts()

	var results []Decomposition
	if d, ok := detectChiitoitsu(counts); ok {
		results = append(results, d)
	}
	if d, ok := detectKokushi(counts); ok {
		results = append(results, d)
	}
	results = append(results, detectStandard(counts)...)
	return results
}

// IsWinning reports whether h forms any valid agari shape.
func IsWinning(h Hand) bool {
	return len(Decompose(h)) > 0
}

func detectChiitoitsu(counts [tile.TileCount]int) (Decomposition, bool) {
	pairs := make([]Meld, 0, 7)
	for id := range uint8(tile.TileCount) {
		switch counts[id] {
		case 0:
			// fine
		case 2:
			pairs = append(pairs, Meld{Kind: MeldPair, Base: id})
		default:
			return Decomposition{}, false
		}
	}
	if len(pairs) != 7 {
		return Decomposition{}, false
	}
	return Decomposition{Form: FormChiitoitsu, Melds: pairs}, true
}

func detectKokushi(counts [tile.TileCount]int) (Decomposition, bool) {
	yaochu := YaochuhaiTiles()
	yaochuSet := map[uint8]bool{}
	for _, id := range yaochu {
		yaochuSet[id] = true
	}
	for id := range uint8(tile.TileCount) {
		if !yaochuSet[id] && counts[id] != 0 {
			return Decomposition{}, false
		}
	}
	var pairTile uint8
	pairFound := false
	for _, id := range yaochu {
		switch counts[id] {
		case 0:
			return Decomposition{}, false
		case 1:
			// fine
		case 2:
			if pairFound {
				return Decomposition{}, false
			}
			pairTile = id
			pairFound = true
		default:
			return Decomposition{}, false
		}
	}
	if !pairFound {
		return Decomposition{}, false
	}
	return Decomposition{
		Form:  FormKokushi,
		Melds: []Meld{{Kind: MeldPair, Base: pairTile}},
	}, true
}

func detectStandard(counts [tile.TileCount]int) []Decomposition {
	var results []Decomposition
	for pairID := range uint8(tile.TileCount) {
		if counts[pairID] < 2 {
			continue
		}
		counts[pairID] -= 2
		acc := []Meld{{Kind: MeldPair, Base: pairID}}
		for _, sets := range extractSets(counts, nil, 0, 4) {
			full := make([]Meld, 0, 5)
			full = append(full, acc...)
			full = append(full, sets...)
			results = append(results, Decomposition{Form: FormStandard, Melds: full})
		}
		counts[pairID] += 2
	}
	return results
}

// extractSets enumerates every way to partition counts into exactly `need`
// sequences/triplets, returning each successful partition as a slice of
// Melds. The recursion always uses the lowest non-zero tile ID, which makes
// the search space finite without explicit deduplication.
func extractSets(counts [tile.TileCount]int, acc []Meld, start uint8, need int) [][]Meld {
	if need == 0 {
		for i := range uint8(tile.TileCount) {
			if counts[i] != 0 {
				return nil
			}
		}
		cp := make([]Meld, len(acc))
		copy(cp, acc)
		return [][]Meld{cp}
	}
	idx := -1
	for i := int(start); i < tile.TileCount; i++ {
		if counts[i] > 0 {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil
	}
	id := uint8(idx)
	var results [][]Meld

	if counts[id] >= 3 {
		counts[id] -= 3
		newAcc := append(append([]Meld{}, acc...), Meld{Kind: MeldTriplet, Base: id})
		results = append(results, extractSets(counts, newAcc, id, need-1)...)
		counts[id] += 3
	}

	if canStartSequence(id) && counts[id] >= 1 && counts[id+1] >= 1 && counts[id+2] >= 1 {
		counts[id]--
		counts[id+1]--
		counts[id+2]--
		newAcc := append(append([]Meld{}, acc...), Meld{Kind: MeldSequence, Base: id})
		results = append(results, extractSets(counts, newAcc, id, need-1)...)
		counts[id]++
		counts[id+1]++
		counts[id+2]++
	}

	return results
}
