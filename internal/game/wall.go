// Package game implements the riichi mahjong game state machine: wall
// construction, dealing, turn cycle, call resolution, bot decisions, and the
// event log consumed by the TUI and golden-game tests.
//
// The package depends on internal/riichi/{tile,hand,yaku,calc} for rules
// evaluation but MUST NOT import internal/play, cmd/, bubbletea, or lipgloss.
// This keeps the game loop testable as plain `go test` and allows future
// changes (network play, headless harnesses) to embed the same engine.
package game

import (
	"math/rand/v2"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

// numSeats is the four-player riichi convention.
const numSeats = 4

// deadWallSize is the standard 14-tile king's wall reserved for kan
// replacements and dora indicators. Live wall after dealing four 13-tile
// hands is 136 − 52 − 14 = 70 tiles.
const deadWallSize = 14

// Wall is a 136-tile shuffled stack with explicit live and dead regions.
// The PRNG is exposed via Rand() so the same seed drives both shuffle and bot
// probabilistic decisions, satisfying the deterministic-shuffle requirement.
type Wall struct {
	tiles     []tile.Tile
	drawIndex int
	rng       *rand.Rand

	// kansDrawn counts the number of rinshan replacement tiles consumed
	// (max 4 per round per riichi rules).
	kansDrawn int

	// kanDoraRevealed counts how many kan-dora indicators have been
	// surfaced (max 4 — one per kan).
	kanDoraRevealed int
}

// WallOptions configures wall construction. Akadora (default in NewWall)
// substitutes one of the four copies of each five-rank tile with the red
// variant before the shuffle, so per-seed reproducibility is preserved.
type WallOptions struct {
	Akadora bool
}

// NewWall builds a 136-tile wall (4 of each of 34 tile types) with akadora
// enabled — one of each five (5m, 5p, 5s) is the red variant. It delegates to
// NewWallWithOptions; callers needing akadora-off should use that directly.
func NewWall(seed int64) *Wall {
	return NewWallWithOptions(seed, WallOptions{Akadora: true})
}

// NewWallWithOptions builds a 136-tile wall and shuffles it using a PCG PRNG
// seeded from `seed`. When opts.Akadora is true, the first encountered copy
// of each five-rank tile is flipped to Red BEFORE the shuffle so two calls
// with the same seed produce byte-identical orders (red fives included).
func NewWallWithOptions(seed int64, opts WallOptions) *Wall {
	tiles := make([]tile.Tile, 0, 34*4)
	for id := range uint8(tile.TileCount) {
		for range 4 {
			tiles = append(tiles, tile.Tile{ID: id})
		}
	}
	if opts.Akadora {
		for _, fiveID := range []uint8{tile.M5, tile.P5, tile.S5} {
			for i := range tiles {
				if tiles[i].ID == fiveID && !tiles[i].Red {
					tiles[i].Red = true
					break
				}
			}
		}
	}
	r := rand.New(rand.NewPCG(uint64(seed), uint64(seed)^0x9E3779B97F4A7C15))
	r.Shuffle(len(tiles), func(i, j int) {
		tiles[i], tiles[j] = tiles[j], tiles[i]
	})
	return &Wall{tiles: tiles, rng: r}
}

// Rand returns the same PRNG used to shuffle the wall, so bot decisions
// reproduce deterministically alongside the shuffle.
func (w *Wall) Rand() *rand.Rand { return w.rng }

// allTiles returns the full 136-tile sequence in wall order. Test-only.
func (w *Wall) allTiles() []tile.Tile {
	out := make([]tile.Tile, len(w.tiles))
	copy(out, w.tiles)
	return out
}

// Deal removes 4×13 tiles from the wall front and returns one hand per seat
// (East, South, West, North). It also reveals the first dora indicator from
// the dead wall. The remaining 70 tiles form the live wall, drawable via
// Draw(); the dead wall (14 tiles, last in the slice) is reserved.
type DealResult struct {
	Hands         [numSeats][]tile.Tile
	DoraIndicator tile.Tile
}

func (w *Wall) Deal() DealResult {
	var hands [numSeats][]tile.Tile
	for seat := range numSeats {
		hands[seat] = make([]tile.Tile, 13)
		for i := range 13 {
			hands[seat][i] = w.tiles[w.drawIndex]
			w.drawIndex++
		}
	}
	// First dora indicator is the third tile from the top of the dead wall —
	// for v1 we just take the last tile of the wall as a stable proxy. Real
	// riichi positions it at a specific dead-wall slot; this simplification
	// stays correct for "single dora indicator revealed at deal time".
	indicator := w.tiles[len(w.tiles)-1]
	return DealResult{Hands: hands, DoraIndicator: indicator}
}

// Draw pops one tile from the front of the live wall. Returns ok=false once
// the live wall is exhausted (dead wall is reserved and never draws via Draw).
func (w *Wall) Draw() (tile.Tile, bool) {
	live := len(w.tiles) - deadWallSize
	if w.drawIndex >= live {
		return tile.Tile{}, false
	}
	t := w.tiles[w.drawIndex]
	w.drawIndex++
	return t, true
}

// LiveRemaining returns the number of tiles still drawable from the live wall.
func (w *Wall) LiveRemaining() int {
	live := len(w.tiles) - deadWallSize
	return live - w.drawIndex
}

// maxKansPerRound is the standard riichi cap on kan declarations per hand.
const maxKansPerRound = 4

// RinshanDraw pulls a replacement tile from the dead wall after a kan
// declaration. The tile is taken from a fixed slot reserved for rinshan
// (does not consume from the live wall — `LiveRemaining()` is unchanged).
// Returns ok=false when 4 kans have already been drawn this round (max
// per riichi rules).
//
// Layout: rinshan slot k (0-indexed) sits at `tiles[len-2-2*k]`. Slot 0
// is one position to the left of the initial dora indicator (`tiles[len-1]`).
func (w *Wall) RinshanDraw() (tile.Tile, bool) {
	if w.kansDrawn >= maxKansPerRound {
		return tile.Tile{}, false
	}
	idx := len(w.tiles) - 2 - 2*w.kansDrawn
	t := w.tiles[idx]
	w.kansDrawn++
	return t, true
}

// RevealKanDora exposes the next kan-dora indicator from the dead wall.
// Called by Game.afterKan after a successful rinshan draw. The indicator
// is appended to the game's `doraIndicators` list and counted toward dora
// han at win time.
//
// Layout: kan-dora slot k (0-indexed) sits at `tiles[len-3-2*k]`, two
// positions to the left of the corresponding rinshan slot.
func (w *Wall) RevealKanDora() tile.Tile {
	idx := len(w.tiles) - 3 - 2*w.kanDoraRevealed
	t := w.tiles[idx]
	w.kanDoraRevealed++
	return t
}
