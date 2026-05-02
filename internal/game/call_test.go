package game

import (
	"testing"

	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

func TestResolveClaimsNoClaimsReturnsNotOK(t *testing.T) {
	_, _, ok := ResolveClaims(nil, SeatEast)
	if ok {
		t.Errorf("ResolveClaims(nil) ok = true, want false")
	}
	_, _, ok = ResolveClaims(map[Seat]Claim{}, SeatEast)
	if ok {
		t.Errorf("ResolveClaims(empty) ok = true, want false")
	}
}

func TestResolveClaimsAllPassReturnsNotOK(t *testing.T) {
	claims := map[Seat]Claim{
		SeatSouth: {Kind: ClaimPass},
		SeatWest:  {Kind: ClaimPass},
		SeatNorth: {Kind: ClaimPass},
	}
	_, _, ok := ResolveClaims(claims, SeatEast)
	if ok {
		t.Errorf("ResolveClaims(all-pass) ok = true, want false")
	}
}

func TestResolveClaimsRonBeatsPonAndChi(t *testing.T) {
	claims := map[Seat]Claim{
		SeatSouth: {Kind: ClaimChi},
		SeatWest:  {Kind: ClaimPon},
		SeatNorth: {Kind: ClaimRon},
	}
	winner, kind, ok := ResolveClaims(claims, SeatEast)
	if !ok {
		t.Fatalf("ResolveClaims ok = false, want true")
	}
	if winner != SeatNorth {
		t.Errorf("winner = %d, want %d (North)", winner, SeatNorth)
	}
	if kind != ClaimRon {
		t.Errorf("kind = %d, want ClaimRon", kind)
	}
}

func TestResolveClaimsPonBeatsChi(t *testing.T) {
	claims := map[Seat]Claim{
		SeatSouth: {Kind: ClaimChi},
		SeatWest:  {Kind: ClaimPon},
	}
	winner, kind, ok := ResolveClaims(claims, SeatEast)
	if !ok {
		t.Fatalf("ResolveClaims ok = false, want true")
	}
	if winner != SeatWest {
		t.Errorf("winner = %d, want %d (West)", winner, SeatWest)
	}
	if kind != ClaimPon {
		t.Errorf("kind = %d, want ClaimPon", kind)
	}
}

func TestResolveClaimsKanTiesPonPriority(t *testing.T) {
	// Both pon and open kan share the same priority slot above chi. With one
	// of each present, the resolver picks one — the contract guarantees a kan
	// or pon winner over chi, not which one.
	claims := map[Seat]Claim{
		SeatSouth: {Kind: ClaimChi},
		SeatWest:  {Kind: ClaimKan},
	}
	_, kind, ok := ResolveClaims(claims, SeatEast)
	if !ok {
		t.Fatalf("ResolveClaims ok = false, want true")
	}
	if kind != ClaimKan && kind != ClaimPon {
		t.Errorf("kind = %d, want ClaimKan or ClaimPon (not chi)", kind)
	}
}

func TestResolveClaimsHeadBumpOnCompetingRons(t *testing.T) {
	// East discards. West (distance 2) and North (distance 3) both ron.
	// Head-bump: closer to discarder going right wins → South would be
	// closest, but South is not claiming here. West (distance 2) wins over
	// North (distance 3).
	claims := map[Seat]Claim{
		SeatWest:  {Kind: ClaimRon},
		SeatNorth: {Kind: ClaimRon},
	}
	winner, kind, ok := ResolveClaims(claims, SeatEast)
	if !ok {
		t.Fatalf("ResolveClaims ok = false, want true")
	}
	if winner != SeatWest {
		t.Errorf("winner = %d, want %d (West, closer to discarder)", winner, SeatWest)
	}
	if kind != ClaimRon {
		t.Errorf("kind = %d, want ClaimRon", kind)
	}
}

func TestResolveClaimsHeadBumpClosestSeatWinsFromAnyDiscarder(t *testing.T) {
	// South discards. West (distance 1) and North (distance 2) both ron.
	// West wins by head-bump.
	claims := map[Seat]Claim{
		SeatWest:  {Kind: ClaimRon},
		SeatNorth: {Kind: ClaimRon},
	}
	winner, _, ok := ResolveClaims(claims, SeatSouth)
	if !ok {
		t.Fatalf("ResolveClaims ok = false, want true")
	}
	if winner != SeatWest {
		t.Errorf("winner = %d, want %d (West, closer to South)", winner, SeatWest)
	}
}

func TestCanPonRequiresAtLeastTwoCopiesInHand(t *testing.T) {
	hand := []tile.Tile{
		{ID: tile.M1}, {ID: tile.M1}, {ID: tile.M2}, {ID: tile.M3},
	}
	if !CanPon(hand, tile.Tile{ID: tile.M1}) {
		t.Errorf("CanPon(hand with two 1m, discard 1m) = false, want true")
	}
	if CanPon(hand, tile.Tile{ID: tile.M2}) {
		t.Errorf("CanPon(hand with one 2m, discard 2m) = true, want false")
	}
	if CanPon(hand, tile.Tile{ID: tile.M9}) {
		t.Errorf("CanPon(hand with no 9m, discard 9m) = true, want false")
	}
}

func TestCanChiAllowsOnlyKamichaDiscarder(t *testing.T) {
	// Discarder East, claimant South: South is kamicha of West, not East.
	// Wait — in seat order E→S→W→N, the next-to-act after East is South;
	// kamicha of South IS East. So South CAN chi East's discard.
	//
	// Verify the geometric mapping: kamicha-of-S returns E, so chi-from-S is
	// legal when discarder=E and not when discarder is anything else.
	hand := []tile.Tile{{ID: tile.M2}, {ID: tile.M3}, {ID: tile.M5}}
	disc := tile.Tile{ID: tile.M4}

	if got := CanChi(hand, disc, SeatEast, SeatSouth); len(got) == 0 {
		t.Errorf(
			"CanChi(South claims East's 4m with 2m+3m / 3m+5m) = no options, want at least one",
		)
	}
	// Discarder South, claimant North: kamicha-of-N is W, not S → illegal.
	if got := CanChi(hand, disc, SeatSouth, SeatNorth); len(got) != 0 {
		t.Errorf("CanChi(North claims South's discard) = options, want none (only kamicha can chi)")
	}
}

func TestCanChiReturnsAllLegalSequenceCompletions(t *testing.T) {
	// Hand has 4m, 6m, 7m. Discard is 5m. Legal sequences:
	//   4m + 5m + 6m  (using 4m and 6m)
	//   5m + 6m + 7m  (using 6m and 7m)
	hand := []tile.Tile{{ID: tile.M4}, {ID: tile.M6}, {ID: tile.M7}}
	disc := tile.Tile{ID: tile.M5}

	options := CanChi(hand, disc, SeatEast, SeatSouth)
	if len(options) != 2 {
		t.Fatalf("CanChi options = %d, want 2", len(options))
	}
}

func TestCanChiRejectsHonorTiles(t *testing.T) {
	hand := []tile.Tile{{ID: tile.SouthWind}, {ID: tile.WestWind}}
	disc := tile.Tile{ID: tile.EastWind}
	if got := CanChi(hand, disc, SeatEast, SeatSouth); len(got) != 0 {
		t.Errorf("CanChi on honor discard = options, want none")
	}
}

func TestCanChiRejectsCrossSuitSequences(t *testing.T) {
	// 4p+6p in hand; discarded 5m — cannot chi (different suits).
	hand := []tile.Tile{{ID: tile.P4}, {ID: tile.P6}}
	disc := tile.Tile{ID: tile.M5}
	if got := CanChi(hand, disc, SeatEast, SeatSouth); len(got) != 0 {
		t.Errorf("CanChi cross-suit = options, want none")
	}
}
