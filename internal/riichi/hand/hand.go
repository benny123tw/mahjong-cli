// Package hand models a riichi mahjong hand and its winning-shape
// decomposition.
//
// A Hand carries 13 or 14 concealed tiles plus contextual flags (open vs
// concealed, win-type, the winning tile). For v1 the CLI surface only ever
// produces fully-concealed hands; the Open flag exists so yaku tests can
// exercise the open-hand han downgrades for honitsu / sanshoku / ittsuu /
// iipeikou / pinfu without a meld model. A future change adds real meld
// parsing alongside game-loop input.
package hand

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

type Hand struct {
	Concealed []tile.Tile
	Open      bool
	Winning   tile.Tile
	IsTsumo   bool
}

func (h Hand) Sorted() []tile.Tile {
	out := make([]tile.Tile, len(h.Concealed))
	copy(out, h.Concealed)
	sort.Slice(out, func(i, j int) bool {
		return tile.Compare(out[i], out[j]) < 0
	})
	return out
}

func (h Hand) Counts() [tile.TileCount]int {
	var c [tile.TileCount]int
	for _, t := range h.Concealed {
		c[t.ID]++
	}
	return c
}

type AgariForm uint8

const (
	FormStandard AgariForm = iota
	FormChiitoitsu
	FormKokushi
)

func (f AgariForm) String() string {
	switch f {
	case FormStandard:
		return "standard"
	case FormChiitoitsu:
		return "chiitoitsu"
	case FormKokushi:
		return "kokushi"
	}
	return "?"
}

type MeldKind uint8

const (
	MeldPair MeldKind = iota
	MeldSequence
	MeldTriplet
)

type Meld struct {
	Kind MeldKind
	Base uint8
}

func (m Meld) Tiles() []uint8 {
	switch m.Kind {
	case MeldPair:
		return []uint8{m.Base, m.Base}
	case MeldSequence:
		return []uint8{m.Base, m.Base + 1, m.Base + 2}
	case MeldTriplet:
		return []uint8{m.Base, m.Base, m.Base}
	}
	return nil
}

func (m Meld) Contains(id uint8) bool {
	return slices.Contains(m.Tiles(), id)
}

// Decomposition is one valid reading of a winning hand.
//   - Standard: Melds[0] is the pair, Melds[1..4] are the four sets.
//   - Chiitoitsu: seven pair melds.
//   - Kokushi: a single pair meld indicating which yaochuhai is doubled.
type Decomposition struct {
	Form  AgariForm
	Melds []Meld
}

func (d Decomposition) Pair() Meld {
	if len(d.Melds) == 0 {
		return Meld{}
	}
	return d.Melds[0]
}

func (d Decomposition) Sets() []Meld {
	if d.Form != FormStandard || len(d.Melds) < 5 {
		return nil
	}
	return d.Melds[1:]
}

// CanonicalString is a stable lexicographic representation used for
// deterministic tie-break in Decomposition Selection.
func (d Decomposition) CanonicalString() string {
	tokens := make([]string, 0, len(d.Melds))
	for _, m := range d.Melds {
		var k byte
		switch m.Kind {
		case MeldPair:
			k = 'P'
		case MeldSequence:
			k = 'C'
		case MeldTriplet:
			k = 'T'
		}
		tokens = append(tokens, fmt.Sprintf("%c%02d", k, m.Base))
	}
	sort.Strings(tokens)
	return fmt.Sprintf("%d:%s", d.Form, joinTokens(tokens))
}

func joinTokens(tokens []string) string {
	var out strings.Builder
	for i, t := range tokens {
		if i > 0 {
			out.WriteString(",")
		}
		out.WriteString(t)
	}
	return out.String()
}

// IsYaochuhai reports whether the tile ID is a terminal (1 or 9 of a numeric
// suit) or any honor — the 13 tiles that participate in kokushi musou.
func IsYaochuhai(id uint8) bool {
	if id >= tile.EastWind {
		return true
	}
	rank := tile.Tile{ID: id}.Rank()
	return rank == 1 || rank == 9
}

// YaochuhaiTiles returns the 13 yaochuhai IDs in canonical order.
func YaochuhaiTiles() [13]uint8 {
	return [13]uint8{
		tile.M1, tile.M9,
		tile.P1, tile.P9,
		tile.S1, tile.S9,
		tile.EastWind, tile.SouthWind, tile.WestWind, tile.NorthWind,
		tile.Haku, tile.Hatsu, tile.Chun,
	}
}

// canStartSequence reports whether a sequence (chii) may begin at this tile.
// Honors cannot form sequences. Numeric tiles can start a sequence only if
// their rank is at most 7 (so 7-8-9 is the highest).
func canStartSequence(id uint8) bool {
	if id >= tile.EastWind {
		return false
	}
	r := tile.Tile{ID: id}.Rank()
	return r <= 7
}
