package cmd

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/benny123tw/mahjong-cli/internal/game"
	"github.com/benny123tw/mahjong-cli/internal/play"
	"github.com/benny123tw/mahjong-cli/internal/riichi/calc"
	"github.com/benny123tw/mahjong-cli/internal/riichi/score"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
	"github.com/benny123tw/mahjong-cli/internal/riichi/yaku"
)

var (
	flagPlayASCII     bool
	flagPlaySeed      int64
	flagPlayNoAkadora bool
	flagPlayDemoEnd   string
)

var playCmd = &cobra.Command{
	Use:   "play",
	Short: "Launch the riichi play screen (TUI)",
	Long: `Launch an interactive bubbletea TUI for a riichi hand against three
dummy bots. Use --seed to reproduce a specific game; without --seed the
program prints the OS-derived seed at startup so the run can be replayed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var renderer play.Renderer = play.UnicodeRenderer{}
		if flagPlayASCII {
			renderer = play.ASCIIRenderer{}
		}

		seed := flagPlaySeed
		if seed == 0 {
			seed = randomSeed()
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Seed: %d\n", seed)

		var match *game.Match
		if flagPlayNoAkadora {
			match = game.NewMatchWithOptions(seed, game.MatchOptions{Akadora: false})
		} else {
			match = game.NewMatch(seed)
		}
		if flagPlayDemoEnd != "" {
			if err := setupDemoEnd(match, flagPlayDemoEnd); err != nil {
				return err
			}
		}
		m := play.NewWithMatch(renderer, match)
		_, err := tea.NewProgram(m).Run()
		return err
	},
}

// setupDemoEnd plants test hands and a StateRoundOver outcome on the
// match's current game so the play screen boots directly into the
// end-of-hand reveal panel. Useful for visually inspecting the panel
// without driving a full game to termination. Supported kinds: ron,
// tsumo, chankan, ryuukyoku.
func setupDemoEnd(m *game.Match, kind string) error {
	cur := m.CurrentGame()

	// All four seats get readable face-up hands. South gets a winning
	// pinfu/tanyao shape with a 5p triplet for the ron/tsumo demos.
	cur.SetTestHand(game.SeatEast, []tile.Tile{
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.P1},
		{ID: tile.P2},
		{ID: tile.P3},
		{ID: tile.S6},
		{ID: tile.S7},
		{ID: tile.S8},
		{ID: tile.Haku},
		{ID: tile.Haku},
		{ID: tile.M1},
		{ID: tile.M1},
	})
	cur.SetTestHand(game.SeatSouth, []tile.Tile{
		{ID: tile.M2},
		{ID: tile.M3},
		{ID: tile.M4},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P5},
		{ID: tile.P6},
		{ID: tile.P7},
		{ID: tile.P8},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.M9},
		{ID: tile.M9},
	})
	cur.SetTestHand(game.SeatWest, []tile.Tile{
		{ID: tile.M5},
		{ID: tile.M6},
		{ID: tile.M7},
		{ID: tile.P3},
		{ID: tile.P4},
		{ID: tile.P5},
		{ID: tile.S3},
		{ID: tile.S4},
		{ID: tile.S5},
		{ID: tile.Hatsu},
		{ID: tile.Hatsu},
		{ID: tile.WestWind},
		{ID: tile.WestWind},
	})
	cur.SetTestHand(game.SeatNorth, []tile.Tile{
		{ID: tile.M6},
		{ID: tile.M7},
		{ID: tile.M8},
		{ID: tile.P7},
		{ID: tile.P8},
		{ID: tile.P9},
		{ID: tile.S1},
		{ID: tile.S2},
		{ID: tile.S3},
		{ID: tile.Chun},
		{ID: tile.Chun},
		{ID: tile.NorthWind},
		{ID: tile.NorthWind},
	})
	// Plant an open meld on West to demonstrate the per-seat meld renderer.
	cur.SetTestMeld(game.SeatWest, game.Meld{
		Kind:  game.MeldPon,
		Tiles: []tile.Tile{{ID: tile.Hatsu}, {ID: tile.Hatsu}, {ID: tile.Hatsu}},
		From:  game.SeatNorth,
	})

	winResult := &calc.Result{
		YakuMatches: []yaku.Match{
			{Name: "Riichi", Han: 1},
			{Name: "Pinfu", Han: 1},
			{Name: "Tanyao", Han: 1},
		},
		Han:   3,
		Fu:    30,
		Award: score.Award{Han: 3, Fu: 30, Base: 480, Total: 3900},
	}

	switch kind {
	case "ron":
		cur.SetTestState(game.StateRoundOver{Outcome: game.OutcomeRon{
			Winner: game.SeatSouth,
			Loser:  game.SeatEast,
			Tile:   tile.Tile{ID: tile.P5},
			Result: winResult,
		}})
	case "tsumo":
		cur.SetTestState(game.StateRoundOver{Outcome: game.OutcomeTsumo{
			Winner: game.SeatSouth,
			Tile:   tile.Tile{ID: tile.P5},
			Result: &calc.Result{
				YakuMatches: []yaku.Match{
					{Name: "Riichi", Han: 1},
					{Name: "Menzen tsumo", Han: 1},
					{Name: "Pinfu", Han: 1},
					{Name: "Tanyao", Han: 1},
				},
				Han:   4,
				Fu:    20,
				Award: score.Award{Han: 4, Fu: 20, Base: 640, Total: 5200},
			},
		}})
	case "chankan":
		cur.SetTestState(game.StateRoundOver{Outcome: game.OutcomeRon{
			Winner: game.SeatSouth,
			Loser:  game.SeatNorth,
			Tile:   tile.Tile{ID: tile.NorthWind},
			Result: &calc.Result{
				YakuMatches: []yaku.Match{
					{Name: "Riichi", Han: 1},
					{Name: "Chankan", Han: 1},
				},
				Han:   2,
				Fu:    30,
				Award: score.Award{Han: 2, Fu: 30, Base: 480, Total: 2000},
			},
		}})
	case "ryuukyoku":
		// 2/2 case: South+West tenpai, East+North noten → ±1500 each.
		cur.SetTestState(game.StateRoundOver{Outcome: game.OutcomeRyuukyoku{
			TenpaiPlayers: []game.Seat{game.SeatSouth, game.SeatWest},
		}})
	default:
		return fmt.Errorf(
			"--demo-end: unknown kind %q (want one of: ron, tsumo, chankan, ryuukyoku)",
			kind,
		)
	}
	return nil
}

// randomSeed pulls 8 bytes from crypto/rand and folds them into an int64.
// Used when --seed is not provided so the game is reproducible after the
// fact via the printed seed.
func randomSeed() int64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Last-resort fallback: low-entropy seed based on a short loop. The
		// program still works deterministically; the user can pass --seed
		// instead if they hit this path.
		return 0xdeadbeef
	}
	return int64(
		binary.LittleEndian.Uint64(b[:]),
	) //nolint:gosec // seed is informational, not cryptographic
}

func init() {
	playCmd.Flags().
		BoolVar(&flagPlayASCII, "ascii", false, "Use ASCII boxed tile rendering instead of Unicode glyphs")
	playCmd.Flags().
		Int64Var(&flagPlaySeed, "seed", 0, "Deterministic shuffle seed (0 = OS random; printed at startup)")
	playCmd.Flags().
		BoolVar(&flagPlayNoAkadora, "no-akadora", false, "Disable red fives (akadora). Default is on, matching modern riichi clients.")
	playCmd.Flags().
		StringVar(&flagPlayDemoEnd, "demo-end", "", "Boot directly into the end-of-hand reveal panel for demo/inspection. One of: ron, tsumo, chankan, ryuukyoku.")
	rootCmd.AddCommand(playCmd)
}
