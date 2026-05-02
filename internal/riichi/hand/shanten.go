package hand

import "github.com/benny123tw/mahjong-cli/internal/riichi/tile"

// Shanten returns the number of tile exchanges needed to reach tenpai for a
// 13-tile hand. For a 14-tile winning hand it returns -1; for a 13-tile
// tenpai hand it returns 0; otherwise it returns the minimum across the
// three agari forms (standard, chiitoitsu, kokushi).
func Shanten(h Hand) int {
	if len(h.Concealed) == 14 {
		if IsWinning(h) {
			return -1
		}
		best := 99
		for i := range h.Concealed {
			sub := make([]tile.Tile, 0, 13)
			sub = append(sub, h.Concealed[:i]...)
			sub = append(sub, h.Concealed[i+1:]...)
			s := Shanten(Hand{Concealed: sub})
			if s < best {
				best = s
			}
		}
		return best
	}
	if len(h.Concealed) != 13 {
		return -2
	}
	counts := h.Counts()
	return minInt(
		shantenStandard(counts),
		shantenChiitoitsu(counts),
		shantenKokushi(counts),
	)
}

func shantenStandard(counts [tile.TileCount]int) int {
	best := 99

	sets, partials := maxBlocks(counts, 0)
	if sets+partials > 4 {
		partials = 4 - sets
	}
	s := 8 - 2*sets - partials
	if s < best {
		best = s
	}

	for id := range uint8(tile.TileCount) {
		if counts[id] >= 2 {
			counts[id] -= 2
			sets2, partials2 := maxBlocks(counts, 0)
			if sets2+partials2 > 4 {
				partials2 = 4 - sets2
			}
			s2 := 8 - 2*sets2 - partials2 - 1
			if s2 < best {
				best = s2
			}
			counts[id] += 2
		}
	}
	return best
}

// maxBlocks searches for the (sets, partials) pair maximizing 2*sets + partials
// across all ways of decomposing counts into triplets, sequences, toitsu,
// ryanmen/penchan partials, kanchan partials, and unused floaters.
func maxBlocks(counts [tile.TileCount]int, start uint8) (int, int) {
	idx := -1
	for i := int(start); i < tile.TileCount; i++ {
		if counts[i] > 0 {
			idx = i
			break
		}
	}
	if idx == -1 {
		return 0, 0
	}
	id := uint8(idx)
	bestSets, bestPartials := -1, -1
	consider := func(s, p int) {
		if bestSets < 0 || 2*s+p > 2*bestSets+bestPartials {
			bestSets, bestPartials = s, p
		}
	}

	if counts[id] >= 3 {
		counts[id] -= 3
		s, p := maxBlocks(counts, id)
		consider(s+1, p)
		counts[id] += 3
	}
	if canStartSequence(id) && counts[id+1] > 0 && counts[id+2] > 0 {
		counts[id]--
		counts[id+1]--
		counts[id+2]--
		s, p := maxBlocks(counts, id)
		consider(s+1, p)
		counts[id]++
		counts[id+1]++
		counts[id+2]++
	}
	if counts[id] >= 2 {
		counts[id] -= 2
		s, p := maxBlocks(counts, id)
		consider(s, p+1)
		counts[id] += 2
	}
	if id < tile.EastWind && id+1 < tile.EastWind &&
		(tile.Tile{ID: id}).Suit() == (tile.Tile{ID: id + 1}).Suit() &&
		counts[id+1] > 0 {
		counts[id]--
		counts[id+1]--
		s, p := maxBlocks(counts, id)
		consider(s, p+1)
		counts[id]++
		counts[id+1]++
	}
	if id < tile.EastWind && id+2 < tile.EastWind &&
		(tile.Tile{ID: id}).Suit() == (tile.Tile{ID: id + 2}).Suit() &&
		counts[id+2] > 0 {
		counts[id]--
		counts[id+2]--
		s, p := maxBlocks(counts, id)
		consider(s, p+1)
		counts[id]++
		counts[id+2]++
	}
	counts[id]--
	s, p := maxBlocks(counts, id)
	consider(s, p)
	counts[id]++

	return bestSets, bestPartials
}

func shantenChiitoitsu(counts [tile.TileCount]int) int {
	pairs := 0
	distinct := 0
	for id := range uint8(tile.TileCount) {
		if counts[id] >= 1 {
			distinct++
		}
		if counts[id] >= 2 {
			pairs++
		}
	}
	if pairs > 7 {
		pairs = 7
	}
	deficit := 0
	if distinct < 7 {
		deficit = 7 - distinct
	}
	return 6 - pairs + deficit
}

func shantenKokushi(counts [tile.TileCount]int) int {
	yaochu := YaochuhaiTiles()
	distinct := 0
	hasPair := 0
	for _, id := range yaochu {
		if counts[id] >= 1 {
			distinct++
		}
		if counts[id] >= 2 {
			hasPair = 1
		}
	}
	return 13 - distinct - hasPair
}

func minInt(xs ...int) int {
	m := xs[0]
	for _, x := range xs[1:] {
		if x < m {
			m = x
		}
	}
	return m
}
