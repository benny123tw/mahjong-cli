package game

import (
	"math/rand/v2"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

// Bot is a single-tier opponent: discards by maximum-isolation heuristic,
// pons yakuhai always, pons non-yakuhai 50% when shanten ≤ 2, chis 40% from
// kamicha only, never declares riichi, never declares kan, ron/tsumo
// whenever a yaku-bearing winning shape is reached.
type Bot struct {
	Seat Seat
	Rng  *rand.Rand
}

// PickDiscard returns the index of the tile the bot discards from `hand`.
// The score model: honors max out (no rank → no neighbors); numeric tiles
// lose isolation when other tiles within ±2 ranks of the same suit appear,
// with closer ranks penalized more heavily. Tiebreak: lowest tile ID.
func (b *Bot) PickDiscard(hand []tile.Tile) int {
	if len(hand) == 0 {
		return -1
	}
	best := 0
	bestScore := isolationScore(0, hand[0], hand)
	for i := 1; i < len(hand); i++ {
		s := isolationScore(i, hand[i], hand)
		switch {
		case s > bestScore:
			best, bestScore = i, s
		case s == bestScore && hand[i].ID < hand[best].ID:
			best = i
		}
	}
	return best
}

const honorIsolationFloor = 1000

func isolationScore(selfIdx int, t tile.Tile, hand []tile.Tile) int {
	if t.IsHonor() {
		// Honors lose isolation only by having same-tile copies; otherwise max.
		score := honorIsolationFloor
		for i, x := range hand {
			if i == selfIdx {
				continue
			}
			if x.ID == t.ID {
				score -= 50
			}
		}
		return score
	}
	score := 100
	suit := t.Suit()
	rank := int(t.Rank())
	for i, x := range hand {
		if i == selfIdx {
			continue
		}
		if x.Suit() != suit {
			continue
		}
		diff := int(x.Rank()) - rank
		if diff < 0 {
			diff = -diff
		}
		if diff <= 2 {
			// Closer rank = more penalty: 0=same rank (-3), 1=adjacent (-2), 2=skip-one (-1).
			score -= (3 - diff)
		}
	}
	return score
}

// ShouldPon decides whether to call pon. Yakuhai pon fires unconditionally
// when 2 copies are present; non-yakuhai pon fires at 50% when shanten ≤ 2
// and 2 copies are present. The PRNG is consumed exactly once on the
// non-yakuhai branch to keep golden-game replays deterministic.
func (b *Bot) ShouldPon(hand []tile.Tile, discarded tile.Tile, isYakuhai bool, shanten int) bool {
	if !CanPon(hand, discarded) {
		return false
	}
	if isYakuhai {
		return true
	}
	if shanten > 2 {
		return false
	}
	return b.Rng.Float64() < 0.5
}

// ShouldChi decides whether to call chi on the given discard from the given
// discarder. Chi is only legal from kamicha; with 2 ranks of legal options,
// the bot fires at 40%, picking the first legal option.
func (b *Bot) ShouldChi(hand []tile.Tile, discarded tile.Tile, discarder Seat) ([2]uint8, bool) {
	options := CanChi(hand, discarded, discarder, b.Seat)
	if len(options) == 0 {
		return [2]uint8{}, false
	}
	if b.Rng.Float64() >= 0.4 {
		return [2]uint8{}, false
	}
	return options[0], true
}

// ShouldRiichi is always false in v1 (riichi deferred to add-smart-ai).
func (b *Bot) ShouldRiichi() bool { return false }

// ShouldKan is always false in v1 (kan deferred to add-kan-support).
func (b *Bot) ShouldKan() bool { return false }
