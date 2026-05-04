package play

import (
	"github.com/benny123tw/mahjong-cli/internal/game"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

// handleKan dispatches the K key based on the current game state.
//
// In `AwaitingDiscard{Player: HumanSeat}`: declare ankan (when the human's
// hand has any concealed 4-of-a-kind) or shouminkan (when the human has
// an open MeldPon plus the matching 4th tile in hand). Riichi-declared
// hands cannot kan in v1 — the K key surfaces a footer note and bails.
// When multiple eligible kans exist, the lowest tile ID wins (deterministic
// pick — no menu).
//
// In `AwaitingClaims`: submit a minkan claim when the human's hand contains
// exactly 3 of the discarded tile. Otherwise surface "kan: illegal".
//
// Outside both states, K is a no-op with a footer note.
func (m Model) handleKan() Model {
	if m.game == nil {
		return m
	}
	switch s := m.game.State().(type) {
	case game.StateAwaitingDiscard:
		if s.Player != HumanSeat {
			return m
		}
		if m.game.IsRiichiDeclared(HumanSeat) {
			m.ackText = "kan: not allowed during riichi"
			return m
		}
		hd := m.game.Hand(HumanSeat)
		if id, ok := firstAnkanID(hd); ok {
			if _, err := m.game.Step(game.InputDeclareAnkan{TileID: id}); err != nil {
				m.ackText = "kan: " + err.Error()
				return m
			}
			m.ackText = "ankan declared"
			return m
		}
		if id, ok := firstShouminkanID(hd, m.game.Melds(HumanSeat)); ok {
			if _, err := m.game.Step(game.InputDeclareShouminkan{TileID: id}); err != nil {
				m.ackText = "kan: " + err.Error()
				return m
			}
			m.ackText = "shouminkan declared"
			return m
		}
		m.ackText = "kan: no eligible declaration"
		return m
	case game.StateAwaitingClaims:
		humanHand := m.game.Hand(HumanSeat)
		if !game.CanKan(humanHand, s.Discard) {
			m.ackText = "kan: illegal"
			return m
		}
		_, err := m.game.Step(game.InputResolveClaims{Claims: map[game.Seat]game.Claim{
			HumanSeat: {Kind: game.ClaimKan},
		}})
		if err != nil {
			m.ackText = "kan: " + err.Error()
		}
		return m
	}
	return m
}

// humanKanLegal reports whether the human can declare ankan or shouminkan
// from the current state. True only during AwaitingDiscard{Human} with no
// active riichi and at least one eligible declaration (ankan or shouminkan).
func humanKanLegal(m Model) bool {
	if m.game == nil {
		return false
	}
	s, ok := m.game.State().(game.StateAwaitingDiscard)
	if !ok || s.Player != HumanSeat {
		return false
	}
	if m.game.IsRiichiDeclared(HumanSeat) {
		return false
	}
	hd := m.game.Hand(HumanSeat)
	if _, ok := firstAnkanID(hd); ok {
		return true
	}
	if _, ok := firstShouminkanID(hd, m.game.Melds(HumanSeat)); ok {
		return true
	}
	return false
}

// firstAnkanID returns the lowest tile ID for which the hand contains
// exactly 4 copies. Returns ok=false when no concealed 4-of-a-kind exists.
func firstAnkanID(hand []tile.Tile) (uint8, bool) {
	counts := tileCounts(hand)
	for id := range uint8(tile.TileCount) {
		if counts[id] == 4 {
			return id, true
		}
	}
	return 0, false
}

// firstShouminkanID returns the lowest tile ID for which the player has an
// open MeldPon AND the matching 4th tile in their concealed hand.
func firstShouminkanID(hand []tile.Tile, melds []game.Meld) (uint8, bool) {
	have := map[uint8]bool{}
	for _, t := range hand {
		have[t.ID] = true
	}
	best, found := uint8(0), false
	for _, mld := range melds {
		if mld.Kind != game.MeldPon || len(mld.Tiles) == 0 {
			continue
		}
		id := mld.Tiles[0].ID
		if !have[id] {
			continue
		}
		if !found || id < best {
			best = id
			found = true
		}
	}
	return best, found
}

func tileCounts(hand []tile.Tile) [tile.TileCount]int {
	var counts [tile.TileCount]int
	for _, t := range hand {
		counts[t.ID]++
	}
	return counts
}
