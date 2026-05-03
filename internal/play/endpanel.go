package play

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/benny123tw/mahjong-cli/internal/game"
	"github.com/benny123tw/mahjong-cli/internal/riichi/calc"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

// endPanelFooter is the single-key terminator hint shown in place of the
// action footer while the end-of-hand reveal panel is active. Pressing any
// key advances to the next hand (or to the standings screen on hanchan-end
// or tobi). Wrapped in spec-mandated brackets and em-dash for parity with
// the End-of-Hand Acknowledgement requirement.
const endPanelFooter = "[Any key — Continue]"

// winnerRowPrefix marks the winning seat in the reveal panel. The prefix
// is rendered before the seat label and bold-styled together with the
// label so the winner row is visually distinct from the others.
const winnerRowPrefix = "[W] "

// nonWinnerRowPrefix is whitespace of the same width as winnerRowPrefix so
// non-winner rows stay vertically aligned with the winner row's label.
const nonWinnerRowPrefix = "    "

// renderEndPanel renders the enriched end-of-hand reveal panel for the
// supported outcome variants (ron, tsumo, ryuukyoku). Returns "" when the
// model is not in a hand-end state, when the underlying state is not a
// supported outcome variant, or when there is no pending transition — the
// caller falls back to its existing minimal rendering in those cases.
func (m Model) renderEndPanel() string {
	if m.game == nil || m.pendingTransition == nil {
		return ""
	}
	st, ok := m.game.State().(game.StateRoundOver)
	if !ok {
		return ""
	}
	switch o := st.Outcome.(type) {
	case game.OutcomeRon:
		return m.renderWinPanel(o.Winner, &o.Loser, o.Tile, o.Result)
	case game.OutcomeTsumo:
		return m.renderWinPanel(o.Winner, nil, o.Tile, o.Result)
	case game.OutcomeRyuukyoku:
		return m.renderRyuukyokuPanel(o)
	}
	return ""
}

// renderWinPanel formats the four-row reveal + yaku/han/fu/deltas
// breakdown for a ron or tsumo outcome. When `loser` is non-nil the
// outcome is a ron and the discarder seat is named in the header;
// otherwise the outcome is a tsumo and the header omits the from-seat
// clause. Chankan-ron is detected by the presence of a "Chankan" yaku
// in the result's match list (the engine sets `ctx.Chankan = true` for
// chankan wins so the yaku evaluator emits this match).
func (m Model) renderWinPanel(
	winner game.Seat,
	loser *game.Seat,
	winTile tile.Tile,
	result *calc.Result,
) string {
	header := m.formatWinHeader(winner, loser, winTile, result)
	rows := []string{statusStyle.Render(header), ""}
	rows = append(rows, m.renderRevealRows(winner, winTile)...)
	rows = append(rows, "")
	rows = append(rows, labelStyle.Render(formatYakuList(result)))
	rows = append(rows, labelStyle.Render(formatTotals(result)))
	rows = append(rows, labelStyle.Render(formatDeltasRow(
		m.pendingTransition.Deltas, m.pendingTransition.NewTotals,
	)))
	rows = append(rows, "", liveKeyStyle.Render(endPanelFooter))
	return strings.Join(rows, "\n")
}

// renderRyuukyokuPanel formats the four-row reveal + tenpai/noten labels
// + deltas for an exhaustive draw. Each seat's row gets a `tenpai` or
// `noten` tag based on whether the seat appears in TenpaiPlayers.
func (m Model) renderRyuukyokuPanel(o game.OutcomeRyuukyoku) string {
	tenpai := tenpaiSet(o.TenpaiPlayers)
	rows := []string{statusStyle.Render("RYUUKYOKU"), ""}
	for s := range game.Seat(4) {
		row := m.renderSeatRow(s, false, tile.Tile{})
		tag := "noten"
		if tenpai[s] {
			tag = "tenpai"
		}
		rows = append(rows, row+"  "+labelStyle.Render(tag))
	}
	rows = append(rows, "")
	rows = append(rows, labelStyle.Render(formatDeltasRow(
		m.pendingTransition.Deltas, m.pendingTransition.NewTotals,
	)))
	rows = append(rows, "", liveKeyStyle.Render(endPanelFooter))
	return strings.Join(rows, "\n")
}

func tenpaiSet(seats []game.Seat) [4]bool {
	var set [4]bool
	for _, s := range seats {
		if int(s) < 4 {
			set[s] = true
		}
	}
	return set
}

func (m Model) formatWinHeader(
	winner game.Seat,
	loser *game.Seat,
	winTile tile.Tile,
	result *calc.Result,
) string {
	kind := "RON"
	if loser == nil {
		kind = "TSUMO"
	}
	if isChankanResult(result) {
		kind = "CHANKAN RON"
	}
	tileStr := winTile.String()
	if loser != nil {
		return fmt.Sprintf(
			"%s — %s wins on %s from %s",
			kind, seatLabel(winner), tileStr, seatLabel(*loser),
		)
	}
	return fmt.Sprintf("%s — %s wins on %s", kind, seatLabel(winner), tileStr)
}

func isChankanResult(result *calc.Result) bool {
	if result == nil {
		return false
	}
	for _, y := range result.YakuMatches {
		if y.Name == "Chankan" {
			return true
		}
	}
	return false
}

// renderRevealRows builds the four seat-rows of the win panel. The winner
// row is prefixed with `winnerRowPrefix` and bold-styled; the winner's
// winning tile is also bold-styled within the concealed-hand block.
func (m Model) renderRevealRows(winner game.Seat, winTile tile.Tile) []string {
	rows := make([]string, 0, 4)
	for s := range game.Seat(4) {
		isWinner := s == winner
		var wt tile.Tile
		if isWinner {
			wt = winTile
		}
		rows = append(rows, m.renderSeatRow(s, isWinner, wt))
	}
	return rows
}

// renderSeatRow renders a single seat's row for the reveal panel: prefix
// + label + face-up concealed hand (sorted) + open melds. When isWinner
// is true the prefix and label are bold and the first occurrence of
// `winTile` within the concealed hand is highlighted via focusedTileStyle.
func (m Model) renderSeatRow(s game.Seat, isWinner bool, winTile tile.Tile) string {
	prefix := nonWinnerRowPrefix
	label := seatLabel(s)
	if isWinner {
		prefix = winnerRowPrefix
		label = lipgloss.NewStyle().Bold(true).Render(label)
		prefix = lipgloss.NewStyle().Bold(true).Render(prefix)
	}
	hand := m.renderSeatHand(s, isWinner, winTile)
	melds := m.renderOpenMeldsForSeat(s)
	parts := []string{prefix + label, "  ", hand}
	if melds != "" {
		parts = append(parts, "  ", melds)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// renderSeatHand renders the seat's concealed hand face-up, sorted by
// tile-ID for consistent display across seats. When isWinner is true,
// the first index whose tile-ID matches winTile.ID is highlighted via
// focusedTileStyle to mark the winning tile.
func (m Model) renderSeatHand(s game.Seat, isWinner bool, winTile tile.Tile) string {
	hand := append([]tile.Tile{}, m.game.Hand(s)...)
	sort.Slice(hand, func(i, j int) bool { return hand[i].ID < hand[j].ID })
	winIdx := -1
	if isWinner {
		for i, t := range hand {
			if t.ID == winTile.ID {
				winIdx = i
				break
			}
		}
	}
	cells := make([]string, 0, len(hand))
	for i, t := range hand {
		block := strings.Join(m.renderer.Tile(t), "\n")
		if i == winIdx {
			block = focusedTileStyle.Render(block)
		}
		cells = append(cells, block)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

// formatYakuList builds the compact `Yaku: <name1> <han1> · <name2> <han2>`
// line for the win panel. Returns `Yaku: (none)` only when the result has
// no yaku matches, which is unreachable for a winning hand under riichi
// rules but kept defensively.
func formatYakuList(result *calc.Result) string {
	if result == nil || len(result.YakuMatches) == 0 {
		return "Yaku: (none)"
	}
	parts := make([]string, 0, len(result.YakuMatches))
	for _, y := range result.YakuMatches {
		parts = append(parts, fmt.Sprintf("%s %d", y.Name, y.Han))
	}
	return "Yaku: " + strings.Join(parts, " · ")
}

// formatTotals renders the `Han N · Fu M · Base K` line. Yakuman hands
// have han=0 in `Result.Han` (the tier indicates yakuman); the renderer
// still shows `Han 0` faithfully because the yaku list above carries the
// yakuman name (e.g., `Suuankou yakuman`).
func formatTotals(result *calc.Result) string {
	if result == nil {
		return "Han 0 · Fu 0 · Base 0"
	}
	return fmt.Sprintf("Han %d · Fu %d · Base %d",
		result.Han, result.Fu, result.Award.Base)
}
