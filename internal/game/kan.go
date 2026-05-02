package game

import (
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

// declareAnkan validates and applies a concealed kan declaration. The seat's
// concealed hand must contain exactly 4 tiles matching `tileID`. On success
// the four tiles become a `MeldKan{KanKind: KanAnkan}`; the rinshan flow
// runs and state remains AwaitingDiscard for the same seat.
func (g *Game) declareAnkan(seat Seat, tileID uint8) error {
	count := 0
	for _, t := range g.hands[seat] {
		if t.ID == tileID {
			count++
		}
	}
	if count != 4 {
		return ErrIllegalKan
	}
	consumed := make([]tile.Tile, 0, 4)
	remaining := g.hands[seat][:0]
	for _, t := range g.hands[seat] {
		if t.ID == tileID && len(consumed) < 4 {
			consumed = append(consumed, t)
			continue
		}
		remaining = append(remaining, t)
	}
	g.hands[seat] = remaining
	g.melds[seat] = append(g.melds[seat], Meld{
		Kind:    MeldKan,
		KanKind: KanAnkan,
		Tiles:   consumed,
	})
	g.logf("ankan %s %s", seatName(seat), consumed[0])
	return g.afterKan(seat)
}

// declareMinkan validates and applies an open kan call from `claimant` on
// `discarder`'s `discard`. The claimant must hold exactly 3 matching tiles
// in their concealed hand. On success the 3 tiles + the discard become a
// `MeldKan{KanKind: KanMinkan, From: discarder}`; the discard is popped
// from the discarder's pond, ippatsu windows close, and the rinshan flow
// runs.
func (g *Game) declareMinkan(claimant Seat, discard tile.Tile, discarder Seat) error {
	count := 0
	for _, t := range g.hands[claimant] {
		if t.ID == discard.ID {
			count++
		}
	}
	if count != 3 {
		return ErrIllegalKan
	}
	consumed := make([]tile.Tile, 0, 3)
	remaining := g.hands[claimant][:0]
	for _, t := range g.hands[claimant] {
		if t.ID == discard.ID && len(consumed) < 3 {
			consumed = append(consumed, t)
			continue
		}
		remaining = append(remaining, t)
	}
	g.hands[claimant] = remaining
	tiles := append([]tile.Tile{discard}, consumed...)
	g.melds[claimant] = append(g.melds[claimant], Meld{
		Kind:    MeldKan,
		KanKind: KanMinkan,
		Tiles:   tiles,
		From:    discarder,
	})
	g.popLastDiscard(discarder)
	g.callsHappened = true
	g.closeAllIppatsuWindows()
	g.logf("minkan %s from %s on %s", seatName(claimant), seatName(discarder), discard)
	return g.afterKan(claimant)
}

// declareShouminkan validates a shouminkan upgrade: the seat must already
// have an open `MeldPon` matching `tileID` AND hold the 4th matching tile
// in their concealed hand. On success the engine transitions to
// `StateAwaitingChankan` to give other seats a ron window. The actual
// pon→kan upgrade and rinshan flow happen in `stepFromAwaitingChankan`
// after the chankan window resolves with no ron.
func (g *Game) declareShouminkan(seat Seat, tileID uint8) error {
	hasPon := false
	for _, m := range g.melds[seat] {
		if m.Kind == MeldPon && len(m.Tiles) > 0 && m.Tiles[0].ID == tileID {
			hasPon = true
			break
		}
	}
	if !hasPon {
		return ErrIllegalKan
	}
	hasUpgrade := false
	for _, t := range g.hands[seat] {
		if t.ID == tileID {
			hasUpgrade = true
			break
		}
	}
	if !hasUpgrade {
		return ErrIllegalKan
	}
	g.state = StateAwaitingChankan{
		UpgradeTile: tile.Tile{ID: tileID},
		Declarer:    seat,
	}
	g.logf("shouminkan-pending %s %s", seatName(seat), tile.Tile{ID: tileID})
	return nil
}

// completeShouminkan upgrades the existing pon meld in place to KanShouminkan,
// removes the 4th tile from the declarer's concealed hand, and runs the
// rinshan flow. Called only from `stepFromAwaitingChankan` after the
// chankan window resolves with no ron.
func (g *Game) completeShouminkan(declarer Seat, upgradeTile tile.Tile) error {
	upgraded := false
	for i := range g.melds[declarer] {
		m := &g.melds[declarer][i]
		if m.Kind == MeldPon && len(m.Tiles) > 0 && m.Tiles[0].ID == upgradeTile.ID {
			m.KanKind = KanShouminkan
			m.Tiles = append(m.Tiles, upgradeTile)
			m.Kind = MeldKan
			upgraded = true
			break
		}
	}
	if !upgraded {
		return ErrIllegalKan
	}
	// Remove one matching tile from concealed hand.
	removed := false
	for i, t := range g.hands[declarer] {
		if t.ID == upgradeTile.ID {
			g.hands[declarer] = append(g.hands[declarer][:i], g.hands[declarer][i+1:]...)
			removed = true
			break
		}
	}
	if !removed {
		return ErrIllegalKan
	}
	g.callsHappened = true
	g.closeAllIppatsuWindows()
	g.logf("shouminkan %s %s", seatName(declarer), upgradeTile)
	return g.afterKan(declarer)
}

// afterKan is the shared post-kan flow: pull a rinshan replacement tile,
// reveal an additional kan-dora indicator, set the seat's lastDrawWasRinshan
// flag, and transition to StateAwaitingDiscard for the same seat. Returns
// `ErrIllegalKan` when the rinshan reserve is exhausted (5th kan attempted),
// in which case the caller is responsible for unwinding the meld.
func (g *Game) afterKan(seat Seat) error {
	t, ok := g.wall.RinshanDraw()
	if !ok {
		return ErrIllegalKan
	}
	g.hands[seat] = append(g.hands[seat], t)
	g.lastDrawWasRinshan[seat] = true
	g.doraIndicators = append(g.doraIndicators, g.wall.RevealKanDora())
	g.state = StateAwaitingDiscard{Player: seat}
	g.logf("rinshan %s %s", seatName(seat), t)
	return nil
}
