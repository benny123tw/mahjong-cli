package tile

import "fmt"

// Parse decodes a hand string in standard riichi notation into a slice of
// Tiles. The string is a sequence of <digit><suit> pairs:
//
//	1m..9m   — man tiles (digit 1-9)
//	0m       — red five man (akadora)
//	1p..9p, 0p — pin tiles + red five
//	1s..9s, 0s — sou tiles + red five
//	1z..7z   — honors: 1z=East, 2z=South, 3z=West, 4z=North,
//	           5z=Haku (white dragon), 6z=Hatsu (green dragon),
//	           7z=Chun (red dragon)
//
// Parse rejects:
//   - Strings of odd length
//   - Hands outside 13..14 tiles (concealed-only — melds are passed via
//     separate inputs upstream)
//   - Tile codes outside the valid ranges (e.g., 0z, 8z, 10m via stray suit char)
//   - More than 4 copies of any tile, with red fives counting toward the
//     limit of their underlying tile value
func Parse(s string) ([]Tile, error) {
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("hand string %q has odd length; expected pairs of <digit><suit>", s)
	}
	n := len(s) / 2
	if n < 13 || n > 14 {
		return nil, fmt.Errorf("hand has %d tiles; expected 13 or 14", n)
	}
	tiles := make([]Tile, 0, n)
	for i := 0; i < len(s); i += 2 {
		t, err := parseAt(s, i)
		if err != nil {
			return nil, err
		}
		tiles = append(tiles, t)
	}
	if err := validateCounts(tiles); err != nil {
		return nil, err
	}
	return tiles, nil
}

// ParseOne parses exactly one tile code (used by --dora and --uradora flags).
func ParseOne(s string) (Tile, error) {
	if len(s) != 2 {
		return Tile{}, fmt.Errorf("tile code %q must be exactly 2 chars", s)
	}
	return parseAt(s, 0)
}

func parseAt(s string, pos int) (Tile, error) {
	digit := s[pos]
	suit := s[pos+1]
	if digit < '0' || digit > '9' {
		return Tile{}, fmt.Errorf(
			"invalid token %q at position %d: expected digit",
			s[pos:pos+2],
			pos,
		)
	}
	n := int(digit - '0')
	switch suit {
	case 'm':
		return numericTile(n, M5, M1, suit, pos)
	case 'p':
		return numericTile(n, P5, P1, suit, pos)
	case 's':
		return numericTile(n, S5, S1, suit, pos)
	case 'z':
		if n < 1 || n > 7 {
			return Tile{}, fmt.Errorf(
				"invalid token %q at position %d: honors are 1z..7z",
				string([]byte{digit, suit}),
				pos,
			)
		}
		return Tile{ID: EastWind + uint8(n-1)}, nil
	default:
		return Tile{}, fmt.Errorf(
			"invalid token %q at position %d: unknown suit %q",
			string([]byte{digit, suit}),
			pos,
			string(suit),
		)
	}
}

func numericTile(n int, redID, baseID uint8, suit byte, pos int) (Tile, error) {
	if n == 0 {
		return Tile{ID: redID, Red: true}, nil
	}
	if n >= 1 && n <= 9 {
		return Tile{ID: baseID + uint8(n-1)}, nil
	}
	return Tile{}, fmt.Errorf(
		"invalid token %q at position %d: numeric tiles are 0-9",
		string([]byte{byte('0' + n), suit}),
		pos,
	)
}

func validateCounts(tiles []Tile) error {
	var counts [TileCount]int
	for _, t := range tiles {
		counts[t.ID]++
		if counts[t.ID] > 4 {
			return fmt.Errorf("more than 4 copies of tile %s", Tile{ID: t.ID}.String())
		}
	}
	return nil
}
