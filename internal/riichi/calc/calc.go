// Package calc orchestrates the rules engine: it walks every valid winning
// decomposition, runs yaku detection, computes fu and final score, and picks
// the highest-scoring decomposition with a deterministic lexicographic
// tie-break.
package calc

import (
	"github.com/benny123tw/mahjong-cli/internal/riichi/hand"
	"github.com/benny123tw/mahjong-cli/internal/riichi/score"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
	"github.com/benny123tw/mahjong-cli/internal/riichi/yaku"
)

type Context struct {
	SeatWind  uint8
	RoundWind uint8
	Riichi    bool
	Dora      []tile.Tile
	Uradora   []tile.Tile

	// Group C situational flags forwarded into yaku.Context. The game loop
	// populates these at win-check time. CLI callers (mahjong calc) leave
	// them at zero values, which is correct: the calc CLI scores hands in
	// isolation without game-loop context.
	Ippatsu      bool
	Haitei       bool
	Houtei       bool
	Rinshan      bool
	Chankan      bool
	DoubleRiichi bool
	Tenhou       bool
	Chiihou      bool
}

type Result struct {
	Form          hand.AgariForm
	Decomposition hand.Decomposition
	YakuMatches   []yaku.Match
	DoraHan       int
	Han           int
	Fu            int
	Award         score.Award
}

// Analyze returns the best winning analysis for a 14-tile Hand or nil if the
// hand has no valid winning decomposition or no yaku (a riichi hand must have
// at least one yaku to claim a win).
func Analyze(h hand.Hand, ctx Context) *Result {
	decomps := hand.Decompose(h)
	if len(decomps) == 0 {
		return nil
	}

	var best *Result
	for _, d := range decomps {
		yakuCtx := yaku.Context{
			SeatWind:     ctx.SeatWind,
			RoundWind:    ctx.RoundWind,
			Riichi:       ctx.Riichi,
			Ippatsu:      ctx.Ippatsu,
			Haitei:       ctx.Haitei,
			Houtei:       ctx.Houtei,
			Rinshan:      ctx.Rinshan,
			Chankan:      ctx.Chankan,
			DoubleRiichi: ctx.DoubleRiichi,
			Tenhou:       ctx.Tenhou,
			Chiihou:      ctx.Chiihou,
		}
		matches := yaku.Evaluate(d, h, yakuCtx)
		han, isYakuman := sumHan(matches)
		if han == 0 && !isYakuman {
			continue
		}

		doraHan := countDoraHan(h, ctx)
		totalHan := han + doraHan

		isPinfu := false
		for _, m := range matches {
			if m.Name == "Pinfu" {
				isPinfu = true
				break
			}
		}
		scoreCtx := score.Context{
			SeatWind:  ctx.SeatWind,
			RoundWind: ctx.RoundWind,
			IsPinfu:   isPinfu,
		}
		fu := score.Fu(d, h, scoreCtx)
		isDealer := ctx.SeatWind == tile.EastWind
		award := score.Compute(totalHan, fu, isDealer, h.IsTsumo, isYakuman)

		r := &Result{
			Form:          d.Form,
			Decomposition: d,
			YakuMatches:   matches,
			DoraHan:       doraHan,
			Han:           totalHan,
			Fu:            fu,
			Award:         award,
		}
		if best == nil || isBetter(r, best) {
			best = r
		}
	}
	return best
}

func sumHan(matches []yaku.Match) (int, bool) {
	total := 0
	isYakuman := false
	for _, m := range matches {
		if m.IsYakuman {
			isYakuman = true
		}
		total += m.Han
	}
	return total, isYakuman
}

func countDoraHan(h hand.Hand, ctx Context) int {
	dora := 0
	for _, t := range h.Concealed {
		if t.Red {
			dora++
		}
	}
	for _, ind := range ctx.Dora {
		target := nextDoraTile(ind.ID)
		for _, t := range h.Concealed {
			if t.ID == target {
				dora++
			}
		}
	}
	if ctx.Riichi {
		for _, ind := range ctx.Uradora {
			target := nextDoraTile(ind.ID)
			for _, t := range h.Concealed {
				if t.ID == target {
					dora++
				}
			}
		}
	}
	return dora
}

func nextDoraTile(indID uint8) uint8 {
	if indID >= tile.EastWind && indID <= tile.NorthWind {
		if indID == tile.NorthWind {
			return tile.EastWind
		}
		return indID + 1
	}
	if indID >= tile.Haku && indID <= tile.Chun {
		if indID == tile.Chun {
			return tile.Haku
		}
		return indID + 1
	}
	rank := (tile.Tile{ID: indID}).Rank()
	suit := (tile.Tile{ID: indID}).Suit()
	var base uint8
	switch suit {
	case tile.SuitMan:
		base = tile.M1
	case tile.SuitPin:
		base = tile.P1
	case tile.SuitSou:
		base = tile.S1
	}
	if rank == 9 {
		return base
	}
	return indID + 1
}

func isBetter(a, b *Result) bool {
	if a.Award.Total != b.Award.Total {
		return a.Award.Total > b.Award.Total
	}
	if a.Han != b.Han {
		return a.Han > b.Han
	}
	if a.Fu != b.Fu {
		return a.Fu > b.Fu
	}
	return a.Decomposition.CanonicalString() < b.Decomposition.CanonicalString()
}
