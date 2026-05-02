package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/benny123tw/mahjong-cli/internal/riichi/calc"
	"github.com/benny123tw/mahjong-cli/internal/riichi/hand"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

var (
	flagSeat    string
	flagRound   string
	flagRiichi  bool
	flagTsumo   bool
	flagDora    []string
	flagUradora []string
)

var calcCmd = &cobra.Command{
	Use:   "calc <hand>",
	Short: "Analyze a riichi hand from a tile string",
	Long: `Analyze a 13- or 14-tile riichi hand string and report shanten/machi
or yaku/fu/points. Tile codes: 1m..9m, 1p..9p, 1s..9s, 1z..7z (winds + dragons),
0m/0p/0s for red fives. The last tile of a 14-tile string is treated as the
winning tile.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCalc(cmd.OutOrStdout(), args[0])
	},
}

func init() {
	calcCmd.Flags().
		StringVar(&flagSeat, "seat", "S", "Seat wind: E (dealer), S, W, N (default S — non-dealer)")
	calcCmd.Flags().StringVar(&flagRound, "round", "E", "Round wind: E or S")
	calcCmd.Flags().BoolVar(&flagRiichi, "riichi", false, "Player declared riichi")
	calcCmd.Flags().BoolVar(&flagTsumo, "tsumo", false, "Win by tsumo (default ron)")
	calcCmd.Flags().StringSliceVar(&flagDora, "dora", nil, "Dora indicator (repeatable, e.g., 5p)")
	calcCmd.Flags().
		StringSliceVar(&flagUradora, "uradora", nil, "Ura-dora indicator (repeatable; only counted with --riichi)")
}

func runCalc(out io.Writer, handStr string) error {
	tiles, err := tile.Parse(handStr)
	if err != nil {
		return fmt.Errorf("parse hand: %w", err)
	}
	seatID, err := parseWind(flagSeat)
	if err != nil {
		return fmt.Errorf("--seat: %w", err)
	}
	roundID, err := parseRoundWind(flagRound)
	if err != nil {
		return fmt.Errorf("--round: %w", err)
	}
	dora, err := parseTileFlags(flagDora, "--dora")
	if err != nil {
		return err
	}
	ura, err := parseTileFlags(flagUradora, "--uradora")
	if err != nil {
		return err
	}

	if len(tiles) == 13 {
		return printTenpai(out, tiles)
	}

	winning := tiles[len(tiles)-1]
	h := hand.Hand{
		Concealed: tiles,
		Winning:   winning,
		IsTsumo:   flagTsumo,
	}
	ctx := calc.Context{
		SeatWind:  seatID,
		RoundWind: roundID,
		Riichi:    flagRiichi,
		Dora:      dora,
		Uradora:   ura,
	}
	result := calc.Analyze(h, ctx)
	if result == nil {
		fmt.Fprintln(out, "Shanten: -1 (winning shape)")
		if !hand.IsWinning(h) {
			fmt.Fprintln(out, "Hand is not a winning shape.")
			return nil
		}
		fmt.Fprintln(out, "Yaku: (none — yakuless win is not allowed)")
		return nil
	}
	return printResult(out, result)
}

func printTenpai(out io.Writer, tiles []tile.Tile) error {
	h := hand.Hand{Concealed: tiles}
	s := hand.Shanten(h)
	m := hand.Machi(h)
	fmt.Fprintf(out, "Shanten: %d\n", s)
	if len(m) > 0 {
		names := make([]string, len(m))
		for i, id := range m {
			names[i] = (tile.Tile{ID: id}).String()
		}
		fmt.Fprintf(out, "Machi: %s\n", strings.Join(names, ", "))
	} else {
		fmt.Fprintln(out, "Machi: (none)")
	}
	return nil
}

func printResult(out io.Writer, r *calc.Result) error {
	fmt.Fprintln(out, "Shanten: -1 (winning)")
	yakuList := make([]string, 0, len(r.YakuMatches)+1)
	for _, m := range r.YakuMatches {
		yakuList = append(yakuList, fmt.Sprintf("%s (%d)", m.Name, m.Han))
	}
	if r.DoraHan > 0 {
		yakuList = append(yakuList, fmt.Sprintf("Dora (%d)", r.DoraHan))
	}
	fmt.Fprintf(out, "Yaku: %s\n", strings.Join(yakuList, ", "))
	fmt.Fprintf(out, "Fu: %d\n", r.Fu)
	fmt.Fprintf(out, "Han: %d  Fu: %d  Points: %d (%s)\n",
		r.Han, r.Fu, r.Award.Total, r.Award.Breakdown)
	return nil
}

func parseWind(s string) (uint8, error) {
	switch strings.ToUpper(s) {
	case "E":
		return tile.EastWind, nil
	case "S":
		return tile.SouthWind, nil
	case "W":
		return tile.WestWind, nil
	case "N":
		return tile.NorthWind, nil
	}
	return 0, fmt.Errorf("invalid wind %q (use E, S, W, N)", s)
}

func parseRoundWind(s string) (uint8, error) {
	switch strings.ToUpper(s) {
	case "E":
		return tile.EastWind, nil
	case "S":
		return tile.SouthWind, nil
	}
	return 0, fmt.Errorf("invalid round wind %q (use E or S)", s)
}

func parseTileFlags(values []string, name string) ([]tile.Tile, error) {
	out := make([]tile.Tile, 0, len(values))
	for _, s := range values {
		t, err := tile.ParseOne(s)
		if err != nil {
			return nil, fmt.Errorf("%s %q: %w", name, s, err)
		}
		out = append(out, t)
	}
	return out, nil
}
