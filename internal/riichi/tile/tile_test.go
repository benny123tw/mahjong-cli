package tile

import "testing"

func TestSuitAndRank(t *testing.T) {
	cases := []struct {
		tile     Tile
		wantSuit Suit
		wantRank uint8
	}{
		{Tile{ID: M1}, SuitMan, 1},
		{Tile{ID: M9}, SuitMan, 9},
		{Tile{ID: P5}, SuitPin, 5},
		{Tile{ID: P5, Red: true}, SuitPin, 5},
		{Tile{ID: S1}, SuitSou, 1},
		{Tile{ID: EastWind}, SuitHonor, 1},
		{Tile{ID: Haku}, SuitHonor, 5},
		{Tile{ID: Chun}, SuitHonor, 7},
	}
	for _, c := range cases {
		if got := c.tile.Suit(); got != c.wantSuit {
			t.Errorf("%s.Suit() = %v, want %v", c.tile, got, c.wantSuit)
		}
		if got := c.tile.Rank(); got != c.wantRank {
			t.Errorf("%s.Rank() = %d, want %d", c.tile, got, c.wantRank)
		}
	}
}

func TestPredicates(t *testing.T) {
	cases := []struct {
		tile        Tile
		isTerminal  bool
		isHonor     bool
		isWind      bool
		isDragon    bool
		isSimple    bool
		isYaochuhai bool
	}{
		{Tile{ID: M1}, true, false, false, false, false, true},
		{Tile{ID: M5}, false, false, false, false, true, false},
		{Tile{ID: M9}, true, false, false, false, false, true},
		{Tile{ID: P2}, false, false, false, false, true, false},
		{Tile{ID: S8}, false, false, false, false, true, false},
		{Tile{ID: EastWind}, false, true, true, false, false, true},
		{Tile{ID: NorthWind}, false, true, true, false, false, true},
		{Tile{ID: Haku}, false, true, false, true, false, true},
		{Tile{ID: Chun}, false, true, false, true, false, true},
	}
	for _, c := range cases {
		if got := c.tile.IsTerminal(); got != c.isTerminal {
			t.Errorf("%s.IsTerminal() = %v, want %v", c.tile, got, c.isTerminal)
		}
		if got := c.tile.IsHonor(); got != c.isHonor {
			t.Errorf("%s.IsHonor() = %v, want %v", c.tile, got, c.isHonor)
		}
		if got := c.tile.IsWind(); got != c.isWind {
			t.Errorf("%s.IsWind() = %v, want %v", c.tile, got, c.isWind)
		}
		if got := c.tile.IsDragon(); got != c.isDragon {
			t.Errorf("%s.IsDragon() = %v, want %v", c.tile, got, c.isDragon)
		}
		if got := c.tile.IsSimple(); got != c.isSimple {
			t.Errorf("%s.IsSimple() = %v, want %v", c.tile, got, c.isSimple)
		}
		if got := c.tile.IsTerminalOrHonor(); got != c.isYaochuhai {
			t.Errorf("%s.IsTerminalOrHonor() = %v, want %v", c.tile, got, c.isYaochuhai)
		}
	}
}

func TestString(t *testing.T) {
	cases := []struct {
		tile Tile
		want string
	}{
		{Tile{ID: M1}, "1m"},
		{Tile{ID: P5}, "5p"},
		{Tile{ID: P5, Red: true}, "0p"},
		{Tile{ID: EastWind}, "1z"},
		{Tile{ID: Chun}, "7z"},
	}
	for _, c := range cases {
		if got := c.tile.String(); got != c.want {
			t.Errorf("Tile{%d, %v}.String() = %q, want %q", c.tile.ID, c.tile.Red, got, c.want)
		}
	}
}

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b Tile
		want int
	}{
		{Tile{ID: M1}, Tile{ID: M2}, -1},
		{Tile{ID: M2}, Tile{ID: M1}, 1},
		{Tile{ID: P5}, Tile{ID: P5}, 0},
		{Tile{ID: P5}, Tile{ID: P5, Red: true}, -1},
		{Tile{ID: P5, Red: true}, Tile{ID: P5}, 1},
		{Tile{ID: EastWind}, Tile{ID: M9}, 1},
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%s, %s) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
