package play

import "github.com/benny123tw/mahjong-cli/internal/riichi/tile"

// vs15 is the U+FE0E text-variation selector, appended to mahjong glyphs to
// force monochrome presentation. Without it, terminals that default to
// emoji-color rendering produce inconsistent cell widths and break layout.
const vs15 = "︎"

// Renderer produces the visual representation of a tile as a slice of lines.
// All tiles produced by a single Renderer SHALL have the same Width and Lines.
//
// Implementations are immutable and selected once at startup.
type Renderer interface {
	Tile(t tile.Tile) []string
	Back() []string
	Width() int
	Lines() int
}

// UnicodeRenderer renders tiles using glyphs from the U+1F000 mahjong-tiles
// block. Each glyph is followed by U+FE0E (VS-15) to force monochrome text
// presentation; without it, many terminals render these as color emoji with
// inconsistent cell widths that wreck horizontal layout.
//
// The Unicode block's tile order differs from the engine's tile-ID order, so
// the lookup table reorders explicitly.
type UnicodeRenderer struct{}

// unicodeTileGlyph maps each Tile.ID (0..33) to the corresponding Unicode
// glyph in the U+1F000 block.
//
// Engine ID order: M1..M9, P1..P9, S1..S9, East, South, West, North, Haku,
// Hatsu, Chun. Unicode block order: East..North, Chun, Hatsu, Haku, M1..M9,
// S1..S9, P1..P9. So the table reorders accordingly.
var unicodeTileGlyph = [tile.TileCount]rune{
	0x1F007, 0x1F008, 0x1F009, 0x1F00A, 0x1F00B, 0x1F00C, 0x1F00D, 0x1F00E, 0x1F00F,
	0x1F019, 0x1F01A, 0x1F01B, 0x1F01C, 0x1F01D, 0x1F01E, 0x1F01F, 0x1F020, 0x1F021,
	0x1F010, 0x1F011, 0x1F012, 0x1F013, 0x1F014, 0x1F015, 0x1F016, 0x1F017, 0x1F018,
	0x1F000, 0x1F001, 0x1F002, 0x1F003,
	0x1F006, 0x1F005, 0x1F004,
}

const unicodeBackGlyph rune = 0x1F02B

func (UnicodeRenderer) Tile(t tile.Tile) []string {
	return []string{string(unicodeTileGlyph[t.ID]) + vs15 + " "}
}

func (UnicodeRenderer) Back() []string {
	return []string{string(unicodeBackGlyph) + vs15 + " "}
}

// Width is the "tile slot" cell count: the glyph plus a trailing space so
// adjacent tiles never touch regardless of font.
//
// Unicode property says the U+1F000 block is East Asian Wide (2 cells), but
// real terminals/fonts disagree:
//   - color-emoji fonts (Apple Color Emoji etc.) draw at 2 cells → slot = 3
//   - monochrome fonts (Symbola, Noto Sans Symbols 2) draw at 1 cell → slot = 2
//
// The trailing space is the smallest single-character separator that keeps
// tiles legible in both regimes; we report 3 for layout sizing because the
// EAW-respecting terminal is the upper bound.
func (UnicodeRenderer) Width() int { return 3 }

func (UnicodeRenderer) Lines() int { return 1 }

// ASCIIRenderer renders each tile as a 4-column × 3-row boxed form. The
// inner label uses the tile's canonical string form (e.g., 1m, 5p, 0p, 1z).
type ASCIIRenderer struct{}

func (ASCIIRenderer) Tile(t tile.Tile) []string {
	return []string{
		"┌──┐",
		"│" + t.String() + "│",
		"└──┘",
	}
}

func (ASCIIRenderer) Back() []string {
	return []string{
		"┌──┐",
		"│▓▓│",
		"└──┘",
	}
}

func (ASCIIRenderer) Width() int { return 4 }
func (ASCIIRenderer) Lines() int { return 3 }

// ASCIIPondRenderer is a 4-column × 1-row compact form used inside per-seat
// discard zones when --ascii is active. The full 3-row boxed form would
// overflow the 24-row budget once four zones are rendered alongside the
// player's hand. Compact form: `[1m]`, `[5p]`, `[1z]`.
type ASCIIPondRenderer struct{}

func (ASCIIPondRenderer) Tile(t tile.Tile) []string {
	return []string{"[" + t.String() + "]"}
}

func (ASCIIPondRenderer) Back() []string { return []string{"[##]"} }

func (ASCIIPondRenderer) Width() int { return 4 }
func (ASCIIPondRenderer) Lines() int { return 1 }
