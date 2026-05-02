package game

import (
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

// ResolveClaims picks the winning claim using fixed priority:
// ron > pon = open kan > chi. On competing ron claims, the head-bump rule
// applies — the seat closest to the discarder going right-around-the-table
// (E→S→W→N→E) wins. Returns ok=false if no live (non-pass) claim is present.
func ResolveClaims(claims map[Seat]Claim, discarder Seat) (Seat, ClaimKind, bool) {
	if len(claims) == 0 {
		return 0, ClaimPass, false
	}

	bestRon := Seat(0)
	bestRonDist := numSeats + 1
	hasRon := false
	for seat, c := range claims {
		if c.Kind == ClaimRon {
			d := int((seat + numSeats - discarder) % numSeats)
			if d < bestRonDist {
				bestRonDist = d
				bestRon = seat
				hasRon = true
			}
		}
	}
	if hasRon {
		return bestRon, ClaimRon, true
	}

	for seat, c := range claims {
		if c.Kind == ClaimPon || c.Kind == ClaimKan {
			return seat, c.Kind, true
		}
	}

	for seat, c := range claims {
		if c.Kind == ClaimChi {
			return seat, ClaimChi, true
		}
	}

	return 0, ClaimPass, false
}

// CanPon reports whether a player holding `hand` can legally pon the given
// `discarded` tile — they must hold at least two copies (by ID, ignoring red).
func CanPon(hand []tile.Tile, discarded tile.Tile) bool {
	count := 0
	for _, t := range hand {
		if t.ID == discarded.ID {
			count++
			if count >= 2 {
				return true
			}
		}
	}
	return false
}

// CanChi returns the legal chi-completion options for a player. Each option
// is a [2]uint8 of the two tile IDs the claimant uses from their hand to
// complete a sequence with `discarded`. Honors and cross-suit combinations
// are rejected. Chi is only legal from kamicha (the seat to the discarder's
// right in turn order); when `claimant` is not the discarder's kamicha, an
// empty slice is returned even if the hand could form a sequence.
func CanChi(hand []tile.Tile, discarded tile.Tile, discarder, claimant Seat) [][2]uint8 {
	if discarded.IsHonor() {
		return nil
	}
	if discarder.Next() != claimant {
		return nil
	}

	id := discarded.ID
	suit := discarded.Suit()
	rank := discarded.Rank()

	have := make(map[uint8]bool, len(hand))
	for _, t := range hand {
		if t.Suit() == suit {
			have[t.ID] = true
		}
	}

	var options [][2]uint8
	// Discard at position 3: need (id-2, id-1) — only when rank ≥ 3.
	if rank >= 3 && have[id-2] && have[id-1] {
		options = append(options, [2]uint8{id - 2, id - 1})
	}
	// Discard at middle: need (id-1, id+1) — only when rank ∈ [2,8].
	if rank >= 2 && rank <= 8 && have[id-1] && have[id+1] {
		options = append(options, [2]uint8{id - 1, id + 1})
	}
	// Discard at start: need (id+1, id+2) — only when rank ≤ 7.
	if rank <= 7 && have[id+1] && have[id+2] {
		options = append(options, [2]uint8{id + 1, id + 2})
	}

	return options
}
