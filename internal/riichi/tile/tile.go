// Package tile defines the riichi mahjong tile model.
//
// A Tile is a uint8 ID in the range [0, 34) plus a separate red-five flag.
// IDs are laid out as 9 man (0..8), 9 pin (9..17), 9 sou (18..26), and
// 7 honors (27..33: East, South, West, North, Haku, Hatsu, Chun).
// Red fives are 5m/5p/5s with Red=true; they participate in every shape
// rule as the underlying 5 and are scored separately as akadora.
package tile

type Suit uint8

const (
	SuitMan Suit = iota
	SuitPin
	SuitSou
	SuitHonor
)

func (s Suit) String() string {
	switch s {
	case SuitMan:
		return "m"
	case SuitPin:
		return "p"
	case SuitSou:
		return "s"
	case SuitHonor:
		return "z"
	}
	return "?"
}

const (
	M1 uint8 = iota
	M2
	M3
	M4
	M5
	M6
	M7
	M8
	M9
	P1
	P2
	P3
	P4
	P5
	P6
	P7
	P8
	P9
	S1
	S2
	S3
	S4
	S5
	S6
	S7
	S8
	S9
	EastWind
	SouthWind
	WestWind
	NorthWind
	Haku
	Hatsu
	Chun
)

const TileCount = 34

type Tile struct {
	ID  uint8
	Red bool
}

func New(id uint8) Tile    { return Tile{ID: id} }
func NewRed(id uint8) Tile { return Tile{ID: id, Red: true} }

func (t Tile) Suit() Suit {
	switch {
	case t.ID < P1:
		return SuitMan
	case t.ID < S1:
		return SuitPin
	case t.ID < EastWind:
		return SuitSou
	default:
		return SuitHonor
	}
}

func (t Tile) Rank() uint8 {
	switch t.Suit() {
	case SuitMan:
		return t.ID - M1 + 1
	case SuitPin:
		return t.ID - P1 + 1
	case SuitSou:
		return t.ID - S1 + 1
	case SuitHonor:
		return t.ID - EastWind + 1
	}
	return 0
}

func (t Tile) IsHonor() bool  { return t.ID >= EastWind }
func (t Tile) IsWind() bool   { return t.ID >= EastWind && t.ID <= NorthWind }
func (t Tile) IsDragon() bool { return t.ID >= Haku && t.ID <= Chun }

func (t Tile) IsTerminal() bool {
	if t.IsHonor() {
		return false
	}
	r := t.Rank()
	return r == 1 || r == 9
}

func (t Tile) IsTerminalOrHonor() bool { return t.IsTerminal() || t.IsHonor() }
func (t Tile) IsSimple() bool          { return !t.IsTerminalOrHonor() }

// String returns the canonical text form: "1m", "9p", "5s", "0p" for red five
// pin, "1z" for East, "7z" for Red dragon.
func (t Tile) String() string {
	s := t.Suit().String()
	if t.Red {
		return "0" + s
	}
	r := t.Rank()
	return string(rune('0'+r)) + s
}

// Compare returns -1, 0, 1 by ID first, then non-red before red of same ID.
func Compare(a, b Tile) int {
	switch {
	case a.ID < b.ID:
		return -1
	case a.ID > b.ID:
		return 1
	case !a.Red && b.Red:
		return -1
	case a.Red && !b.Red:
		return 1
	}
	return 0
}
