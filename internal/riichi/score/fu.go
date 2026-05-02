// Package score computes fu and final points for a winning hand. Fu depends
// on the chosen decomposition, so callers pass a single Decomposition along
// with the Hand and Context — the calc orchestrator is responsible for
// enumerating decompositions and picking the highest-scoring one.
package score

import (
	"github.com/benny123tw/mahjong-cli/internal/riichi/hand"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

type Context struct {
	SeatWind  uint8
	RoundWind uint8
	IsPinfu   bool
}

// Fu returns fu for a winning decomposition. Chiitoitsu is flat 25.
// Kokushi (yakuman) returns 0 since fu is unused at yakuman tier.
// Pinfu-tsumo is flat 20. All other standard hands round up to the nearest 10.
func Fu(d hand.Decomposition, h hand.Hand, ctx Context) int {
	if d.Form == hand.FormChiitoitsu {
		return 25
	}
	if d.Form == hand.FormKokushi {
		return 0
	}
	if ctx.IsPinfu && h.IsTsumo {
		return 20
	}

	fu := 20
	if !h.IsTsumo && !h.Open {
		fu += 10
	}
	if h.IsTsumo && !ctx.IsPinfu {
		fu += 2
	}

	concealed := !h.Open

	for _, m := range d.Sets() {
		if m.Kind != hand.MeldTriplet {
			continue
		}
		isYaochu := (tile.Tile{ID: m.Base}).IsTerminalOrHonor()
		switch {
		case concealed && isYaochu:
			fu += 8
		case concealed && !isYaochu:
			fu += 4
		case !concealed && isYaochu:
			fu += 4
		case !concealed && !isYaochu:
			fu += 2
		}
	}

	if isYakuhaiTile(d.Pair().Base, ctx) {
		fu += 2
	}

	fu += waitShapeFu(d, h.Winning)

	if h.Open && fu == 20 {
		fu += 2
	}

	if fu%10 != 0 {
		fu = ((fu / 10) + 1) * 10
	}
	return fu
}

func waitShapeFu(d hand.Decomposition, winning tile.Tile) int {
	bestFu := -1
	consider := func(v int) {
		if bestFu < 0 || v > bestFu {
			bestFu = v
		}
	}

	for _, m := range d.Sets() {
		if m.Kind != hand.MeldSequence {
			continue
		}
		if !m.Contains(winning.ID) {
			continue
		}
		baseRank := (tile.Tile{ID: m.Base}).Rank()
		pos := winning.ID - m.Base
		switch pos {
		case 0:
			if baseRank == 7 {
				consider(2)
			} else {
				consider(0)
			}
		case 1:
			consider(2)
		case 2:
			if baseRank == 1 {
				consider(2)
			} else {
				consider(0)
			}
		}
	}

	if d.Pair().Base == winning.ID {
		consider(2)
	}

	if bestFu < 0 {
		return 0
	}
	return bestFu
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
