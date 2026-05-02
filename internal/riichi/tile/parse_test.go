package tile

import (
	"strings"
	"testing"
)

// TestParse_TileMappingExample exercises the example tile-mapping table
// from specs/hand-calculator/spec.md.
func TestParse_TileMappingExample(t *testing.T) {
	cases := []struct {
		input    string
		wantID   uint8
		wantRed  bool
		wantSuit Suit
		wantRank uint8
	}{
		{"1m", M1, false, SuitMan, 1},
		{"9m", M9, false, SuitMan, 9},
		{"5p", P5, false, SuitPin, 5},
		{"0p", P5, true, SuitPin, 5},
		{"1z", EastWind, false, SuitHonor, 1},
		{"5z", Haku, false, SuitHonor, 5},
		{"7z", Chun, false, SuitHonor, 7},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			tt, err := ParseOne(c.input)
			if err != nil {
				t.Fatalf("ParseOne(%q) err=%v", c.input, err)
			}
			if tt.ID != c.wantID {
				t.Errorf("ID=%d want %d", tt.ID, c.wantID)
			}
			if tt.Red != c.wantRed {
				t.Errorf("Red=%v want %v", tt.Red, c.wantRed)
			}
			if tt.Suit() != c.wantSuit {
				t.Errorf("Suit=%v want %v", tt.Suit(), c.wantSuit)
			}
			if tt.Rank() != c.wantRank {
				t.Errorf("Rank=%d want %d", tt.Rank(), c.wantRank)
			}
		})
	}
}

func TestParse_FullWinningHand(t *testing.T) {
	tiles, err := Parse("1m2m3m4p5p6p7s8s9s1z1z2z2z2z")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(tiles) != 14 {
		t.Fatalf("got %d tiles, want 14", len(tiles))
	}
	wantIDs := []uint8{
		M1,
		M2,
		M3,
		P4,
		P5,
		P6,
		S7,
		S8,
		S9,
		EastWind,
		EastWind,
		SouthWind,
		SouthWind,
		SouthWind,
	}
	for i, want := range wantIDs {
		if tiles[i].ID != want {
			t.Errorf("tile[%d].ID = %d, want %d", i, tiles[i].ID, want)
		}
		if tiles[i].Red {
			t.Errorf("tile[%d] should not be red", i)
		}
	}
}

func TestParse_TenpaiThirteenTiles(t *testing.T) {
	tiles, err := Parse("1m2m3m4p5p6p7s8s9s1z1z2z2z")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(tiles) != 13 {
		t.Errorf("got %d tiles, want 13", len(tiles))
	}
}

func TestParse_RejectInvalidTokens(t *testing.T) {
	cases := []string{
		"0z" + strings.Repeat("1m", 13), // 0z is not a valid honor
		"8z" + strings.Repeat("1m", 13), // 8z is not a valid honor
		"1x" + strings.Repeat("1m", 13), // unknown suit
		"a1" + strings.Repeat("1m", 13), // non-digit leading char
	}
	for _, in := range cases {
		t.Run(in[:2], func(t *testing.T) {
			if _, err := Parse(in); err == nil {
				t.Errorf("Parse(%q) should fail", in[:8]+"...")
			}
		})
	}
}

func TestParse_RejectOddLength(t *testing.T) {
	if _, err := Parse("1m2m3"); err == nil {
		t.Error("Parse with odd length should fail")
	}
}

func TestParse_RejectOversize(t *testing.T) {
	// 15 tiles
	if _, err := Parse(
		strings.Repeat("1m", 4) + strings.Repeat("2p", 4) + strings.Repeat("3s", 4) + "1z2z3z",
	); err == nil {
		t.Error("Parse with 15 tiles should fail")
	}
}

func TestParse_RejectUndersize(t *testing.T) {
	// 12 tiles
	if _, err := Parse(strings.Repeat("1m", 12)); err == nil {
		// 12 of same tile is also 5+ count, but the size check happens first.
		t.Error("Parse with 12 tiles should fail")
	}
}

func TestParse_RejectFiveCopies(t *testing.T) {
	// 5 copies of 1m + 9 other tiles = 14 valid-size hand but exceeds tile count
	in := strings.Repeat("1m", 5) + "2m3m4m5m6m7m8m9m1p"
	if _, err := Parse(in); err == nil {
		t.Errorf("Parse(%q) with 5 copies of 1m should fail", in)
	}
}

func TestParse_RedFiveCountsTowardLimit(t *testing.T) {
	// 4 normal 5p + 1 red 5p = 5 copies of 5p; should fail.
	in := "5p5p5p5p0p" + "1m2m3m4m6m7m8m9m1s"
	if _, err := Parse(in); err == nil {
		t.Errorf("Parse(%q) with 4×5p + 0p should fail (5 copies of pin-5)", in)
	}
}

func TestParse_FourCopiesOK(t *testing.T) {
	// 4×1m + remaining 10 tiles = 14
	in := strings.Repeat("1m", 4) + "2m3m4m5m6m7m8m9m1p2p"
	if _, err := Parse(in); err != nil {
		t.Errorf("Parse(%q) should succeed: %v", in, err)
	}
}

func TestParseOne_Length(t *testing.T) {
	if _, err := ParseOne("1"); err == nil {
		t.Error("ParseOne with len 1 should fail")
	}
	if _, err := ParseOne("1mm"); err == nil {
		t.Error("ParseOne with len 3 should fail")
	}
}
