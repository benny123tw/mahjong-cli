// Package play hosts the bubbletea v2 program for the riichi play screen.
//
// Layout split:
//   - play.go      — Model, Init, Update, View
//   - render.go    — Renderer interface plus Unicode and ASCII implementations
//   - pond.go      — per-seat discard zone rendering
//   - keys.go      — Key bindings shown in the action footer
//
// The package depends on internal/game for live state, internal/riichi/calc
// and internal/riichi/hand for engine queries. The flow is single-threaded
// bubbletea: every key press routes through Update, which consults
// game.Game.State() to decide what's legal.
package play

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/benny123tw/mahjong-cli/internal/game"
	"github.com/benny123tw/mahjong-cli/internal/riichi/hand"
	"github.com/benny123tw/mahjong-cli/internal/riichi/tile"
)

// botTickInterval is the wall-clock pause between consecutive bot decisions.
// Slow enough to read the previous discard, quick enough to not feel laggy.
const botTickInterval = 250 * time.Millisecond

// BotTickMsg fires periodically while the game is on a bot seat. The Update
// handler consults game state and dispatches one bot decision per tick.
type BotTickMsg struct{}

const (
	targetWidth  = 80
	targetHeight = 24
)

// peekUnknown is the sentinel `PeekShanten()` returns before the player has
// pressed `?` — the shanten cache is unpopulated.
const peekUnknown = -99

// HumanSeat is the seat the human player occupies. Hard-coded to South for
// v1; future changes can let users pick.
const HumanSeat = game.SeatSouth

// Model holds play-screen state. When a *game.Game pointer is supplied via
// NewWithGame, hand and pond data flow from there; the legacy New(renderer)
// path keeps the static fixture so old callers continue to render.
type Model struct {
	game     *game.Game // nil for legacy fixture mode
	hand     hand.Hand  // fixture fallback
	cursor   int
	width    int
	height   int
	renderer Renderer

	// peek cache for `?` key. Zero values mean "not yet asked".
	peekShanten int
	peekMachi   []uint8
	peekHandLen int

	// transient feedback strings shown in the footer.
	ackKey  string
	ackText string
}

// New constructs a Model with the chosen renderer in fixture mode (no game
// pointer). Pond zones start empty. Retained for incremental migration; new
// callers SHOULD use NewWithGame.
func New(renderer Renderer) Model {
	return Model{
		hand:        fixtureHand(),
		renderer:    renderer,
		peekShanten: peekUnknown,
	}
}

// NewWithGame constructs a Model bound to a live game state machine. All
// hand and pond queries flow through the *game.Game pointer.
func NewWithGame(renderer Renderer, g *game.Game) Model {
	return Model{
		game:        g,
		renderer:    renderer,
		peekShanten: peekUnknown,
	}
}

// Hand returns the active hand tiles for the human player. When backed by a
// game pointer, this reads through Game.Hand(HumanSeat); otherwise it falls
// back to the fixture.
func (m Model) Hand() []tile.Tile {
	if m.game != nil {
		return m.game.Hand(HumanSeat)
	}
	return append([]tile.Tile{}, m.hand.Concealed...)
}

// Pond returns the discard pond for the named seat. Empty when no game is
// attached.
func (m Model) Pond(s game.Seat) []tile.Tile {
	if m.game != nil {
		return m.game.Discards(s)
	}
	return nil
}

// GameState returns the current state from the underlying game, or nil when
// in fixture mode.
func (m Model) GameState() game.State {
	if m.game == nil {
		return nil
	}
	return m.game.State()
}

// PeekShanten returns the cached shanten value for the human's hand. Returns
// peekUnknown until the player has pressed `?`.
func (m Model) PeekShanten() int { return m.peekShanten }

// PeekMachi returns the cached machi tile IDs (waits) for the human's hand.
// Empty until the player has pressed `?`.
func (m Model) PeekMachi() []uint8 { return m.peekMachi }

// AckText returns the transient footer feedback string (e.g., "no yaku —
// cannot win"). Cleared on next non-ack action.
func (m Model) AckText() string { return m.ackText }

// Init returns a bot tick command when the round starts on a bot seat
// (which is always the case in v1 — East draws first and the human is
// South). No-op when running in fixture mode without a game.
//
// autoDrawHuman is called for its side-effect on the underlying *game.Game
// pointer; the returned Model copy is intentionally discarded because Init's
// signature returns a Cmd, not a Model.
func (m Model) Init() tea.Cmd {
	m.autoDrawHuman()
	return m.maybeBotTickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case BotTickMsg:
		return m.handleBotTick()
	case tea.KeyPressMsg:
		updated, cmd := m.handleKey(msg.String())
		if cmd != nil {
			return updated, cmd
		}
		mu, _ := updated.(Model)
		mu = mu.autoDrawHuman()
		return mu, mu.maybeBotTickCmd()
	}
	return m, nil
}

// autoDrawHuman fires a Draw input automatically when state lands on
// `AwaitingDraw{HumanSeat}` — the human never has to press a key to draw.
// Standard riichi UX: drawing is automatic, only the discard is a decision.
func (m Model) autoDrawHuman() Model {
	if m.game == nil {
		return m
	}
	if s, ok := m.game.State().(game.StateAwaitingDraw); ok && s.Player == HumanSeat {
		_, _ = m.game.Step(game.InputDraw{})
	}
	return m
}

// maybeBotTickCmd returns a tea.Tick command when the current game state is
// on a bot seat (anything except South in v1). Returns nil otherwise.
func (m Model) maybeBotTickCmd() tea.Cmd {
	if m.game == nil {
		return nil
	}
	if !m.isBotTurn() {
		return nil
	}
	return tea.Tick(botTickInterval, func(time.Time) tea.Msg { return BotTickMsg{} })
}

// isBotTurn reports whether the underlying state expects an action from a
// non-human seat (any seat that isn't HumanSeat).
func (m Model) isBotTurn() bool {
	if m.game == nil {
		return false
	}
	switch s := m.game.State().(type) {
	case game.StateAwaitingDraw:
		return s.Player != HumanSeat
	case game.StateAwaitingDiscard:
		return s.Player != HumanSeat
	case game.StateAwaitingClaims:
		// Claims state involves all non-discarder seats — for simplicity v1
		// auto-passes on bot claims (real bot pon/chi/ron logic ships in
		// task 9.x's full bot dispatch path).
		return true
	}
	return false
}

// handleBotTick advances exactly one bot action and either schedules another
// tick (still bot's turn) or returns to a human turn (no tick).
func (m Model) handleBotTick() (tea.Model, tea.Cmd) {
	if m.game == nil {
		return m, nil
	}
	switch s := m.game.State().(type) {
	case game.StateAwaitingDraw:
		_, _ = m.game.Step(game.InputDraw{})
		_ = s
	case game.StateAwaitingDiscard:
		bot := game.Bot{Seat: s.Player, Rng: m.game.Wall().Rand()}
		idx := max(bot.PickDiscard(m.game.Hand(s.Player)), 0)
		_, _ = m.game.Step(game.InputDiscard{Index: idx})
	case game.StateAwaitingClaims:
		// V1: bots auto-pass. The bot decision logic exists in game.Bot but
		// piping it through here (one decision per non-discarder seat) plus
		// the resolver is its own task; deferred to add-smart-ai.
		_, _ = m.game.Step(game.InputResolveClaims{Claims: nil})
	}
	m = m.autoDrawHuman()
	return m, m.maybeBotTickCmd()
}

func (m Model) handleKey(key string) (tea.Model, tea.Cmd) {
	hand := m.Hand()
	handLen := len(hand)
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "left", "h":
		if m.cursor > 0 {
			m.cursor--
		}
	case "right", "l":
		if m.cursor < handLen-1 {
			m.cursor++
		}
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		n := min(int(key[0]-'0'), handLen)
		if n >= 1 {
			m.cursor = n - 1
		}
	case "?":
		return m.handlePeek(), nil
	case "t":
		return m.handleTsumo(), nil
	case "d", "enter":
		return m.handleDiscard(), nil
	case " ", "space":
		return m.handlePass(), nil
	case "r":
		m.ackText = "riichi: not implemented in v1 (deferred to add-smart-ai)"
		m.ackKey = key
	case "p":
		return m.handlePon(), nil
	case "c":
		return m.handleChi(), nil
	case "k":
		m.ackText = "kan: not supported in v1 (deferred to add-kan-support)"
		m.ackKey = key
	}
	return m, nil
}

func (m Model) handlePeek() Model {
	if m.game == nil {
		return m
	}
	concealed := m.game.Hand(HumanSeat)
	h := hand.Hand{Concealed: concealed}
	if len(concealed) == 13 {
		m.peekShanten = hand.Shanten(h)
		m.peekMachi = hand.Machi(h)
	} else if hand.IsWinning(h) {
		// 14-tile hand — drop-each-tile isn't useful here, so just report
		// shanten of the 14-tile winning check. -1 = winning.
		m.peekShanten = -1
		m.peekMachi = nil
	}
	m.peekHandLen = len(concealed)
	return m
}

func (m Model) handleTsumo() Model {
	if m.game == nil {
		return m
	}
	st, ok := m.game.State().(game.StateAwaitingDiscard)
	if !ok || st.Player != HumanSeat {
		m.ackText = "tsumo: not your turn"
		return m
	}
	_, err := m.game.Step(game.InputDeclareTsumo{})
	if err != nil {
		m.ackText = "no yaku — cannot win"
		return m
	}
	m.ackText = "tsumo!"
	return m
}

func (m Model) handleDiscard() Model {
	if m.game == nil {
		return m
	}
	st, ok := m.game.State().(game.StateAwaitingDiscard)
	if !ok || st.Player != HumanSeat {
		return m
	}
	if _, err := m.game.Step(game.InputDiscard{Index: m.cursor}); err != nil {
		m.ackText = "discard: " + err.Error()
		return m
	}
	m.peekShanten = peekUnknown
	m.peekMachi = nil
	if m.cursor >= len(m.game.Hand(HumanSeat)) {
		m.cursor = max(0, len(m.game.Hand(HumanSeat))-1)
	}
	return m
}

func (m Model) handlePass() Model {
	if m.game == nil {
		return m
	}
	if _, ok := m.game.State().(game.StateAwaitingClaims); !ok {
		return m
	}
	if _, err := m.game.Step(game.InputResolveClaims{Claims: nil}); err != nil {
		m.ackText = "pass: " + err.Error()
	}
	return m
}

func (m Model) handlePon() Model {
	if m.game == nil {
		return m
	}
	cs, ok := m.game.State().(game.StateAwaitingClaims)
	if !ok {
		return m
	}
	if !game.CanPon(m.game.Hand(HumanSeat), cs.Discard) {
		m.ackText = "pon: illegal"
		return m
	}
	_, err := m.game.Step(game.InputResolveClaims{Claims: map[game.Seat]game.Claim{
		HumanSeat: {Kind: game.ClaimPon},
	}})
	if err != nil {
		m.ackText = "pon: " + err.Error()
	}
	return m
}

func (m Model) handleChi() Model {
	if m.game == nil {
		return m
	}
	cs, ok := m.game.State().(game.StateAwaitingClaims)
	if !ok {
		return m
	}
	options := game.CanChi(m.game.Hand(HumanSeat), cs.Discard, cs.Discarder, HumanSeat)
	if len(options) == 0 {
		m.ackText = "chi: illegal"
		return m
	}
	// V1: pick first legal option. A future trainer-aid change exposes a
	// secondary menu when multiple options exist.
	_, err := m.game.Step(game.InputResolveClaims{Claims: map[game.Seat]game.Claim{
		HumanSeat: {Kind: game.ClaimChi, ChiTiles: options[0]},
	}})
	if err != nil {
		m.ackText = "chi: " + err.Error()
	}
	return m
}

// RenderCallFooter renders the call-window footer prompt. Greyed entries are
// illegal calls; live entries are legal claims for the human player.
// Returns empty string when not in claims state.
func (m Model) RenderCallFooter() string {
	if m.game == nil {
		return ""
	}
	cs, ok := m.game.State().(game.StateAwaitingClaims)
	if !ok {
		return ""
	}
	humanHand := m.game.Hand(HumanSeat)
	canPon := game.CanPon(humanHand, cs.Discard)
	canChi := len(game.CanChi(humanHand, cs.Discard, cs.Discarder, HumanSeat)) > 0

	render := func(label string, active bool) string {
		if active {
			return liveKeyStyle.Render(label)
		}
		return greyedKeyStyle.Render(label)
	}
	parts := []string{
		render("[P]on", canPon),
		render("[C]hi", canChi),
		render("[K]an (greyed)", false),
		render("[R]on (greyed)", false),
		liveKeyStyle.Render("[Space] Pass"),
	}
	return strings.Join(parts, "  ")
}

func (m Model) View() tea.View {
	tooSmall := m.width > 0 && (m.width < targetWidth || m.height < targetHeight)
	if tooSmall {
		notice := fmt.Sprintf(
			"Terminal too small — need %d×%d, got %d×%d.\nResize the terminal and try again, or run with --ascii.",
			targetWidth,
			targetHeight,
			m.width,
			m.height,
		)
		return tea.NewView(notice)
	}

	body := m.renderLayout()

	if m.width >= targetWidth && m.height >= targetHeight {
		body = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
	}
	return tea.NewView(body)
}

// fixtureHand is retained for the legacy New(renderer) path that constructs
// a Model without a *game.Game pointer.
func fixtureHand() hand.Hand {
	tiles, err := tile.Parse("1m1m1m4m4m4m7m7m7m9m9m9m5m5m")
	if err != nil {
		panic("play: fixture parse failed: " + err.Error())
	}
	return hand.Hand{
		Concealed: tiles,
		Winning:   tiles[len(tiles)-1],
	}
}

// --- Section renderers below ---

var (
	statusStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Bold(true)
	labelStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	cursorMarkStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	focusedTileStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	greyedKeyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	liveKeyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
)

func (m Model) renderLayout() string {
	parts := []string{
		m.renderStatus(),
		"",
		m.renderToimenRow(),
		"",
		m.renderMidRow(),
		"",
		m.renderPlayerPond(),
		"",
		m.renderHand(),
		"",
		m.renderFooter(),
	}
	return strings.Join(parts, "\n")
}

func (m Model) renderStatus() string {
	wallRemaining := 70
	if m.game != nil {
		wallRemaining = m.game.Wall().LiveRemaining()
	}
	return statusStyle.Render(
		fmt.Sprintf("East 1  ·  Honba 0  ·  Wall %d  ·  Dora: %s  ·  Seat: South  ·  Score 25000",
			wallRemaining, m.firstDoraIndicator()),
	)
}

func (m Model) firstDoraIndicator() string {
	if m.game == nil {
		return "5p"
	}
	d := m.game.DoraIndicators()
	if len(d) == 0 {
		return "?"
	}
	return d[0].String()
}

func (m Model) renderToimenRow() string {
	label := labelStyle.Render("        Toimen — North · 25000")
	row := m.renderBackRow(13)
	pond := renderPondZone(m.Pond(game.SeatNorth), m.renderer)
	if pond == "" {
		return label + "\n" + row
	}
	return label + "\n" + row + "\n" + pond
}

func (m Model) renderBackRow(count int) string {
	cells := make([]string, count)
	back := m.renderer.Back()
	for i := range cells {
		cells[i] = strings.Join(back, "\n")
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

func (m Model) renderMidRow() string {
	kami := m.renderKamichaColumn()
	shimo := m.renderShimochaColumn()
	centre := m.renderCentreInfo()

	kamiStyled := lipgloss.NewStyle().Width(20).Render(kami)
	centreStyled := lipgloss.NewStyle().Width(36).Render(centre)
	shimoStyled := lipgloss.NewStyle().Width(20).Render(shimo)
	return lipgloss.JoinHorizontal(lipgloss.Top, kamiStyled, centreStyled, shimoStyled)
}

func (m Model) renderKamichaColumn() string {
	label := labelStyle.Render("Kamicha · East")
	pond := renderPondZone(m.Pond(game.SeatEast), m.renderer)
	if pond == "" {
		return label
	}
	return label + "\n" + pond
}

func (m Model) renderShimochaColumn() string {
	label := labelStyle.Render("Shimocha · West")
	pond := renderPondZone(m.Pond(game.SeatWest), m.renderer)
	if pond == "" {
		return label
	}
	return label + "\n" + pond
}

func (m Model) renderCentreInfo() string {
	wall := 70
	if m.game != nil {
		wall = m.game.Wall().LiveRemaining()
	}
	return labelStyle.Render(fmt.Sprintf(
		"     Round: East 1\n     Honba:  0\n     Wall:   %d\n     Dora:   %s\n     You:    South · 25000",
		wall,
		m.firstDoraIndicator(),
	))
}

func (m Model) renderPlayerPond() string {
	label := labelStyle.Render("Your discards:")
	pond := renderPondZone(m.Pond(HumanSeat), m.renderer)
	if pond == "" {
		return label
	}
	return label + "\n" + pond
}

func (m Model) renderHand() string {
	tiles := m.Hand()
	cells := make([]string, len(tiles))
	for i, t := range tiles {
		tileLines := m.renderer.Tile(t)
		tileBlock := strings.Join(tileLines, "\n")
		if i == m.cursor {
			tileBlock = focusedTileStyle.Render(tileBlock)
		}
		cells[i] = tileBlock
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

func (m Model) renderFooter() string {
	if m.game != nil {
		if cf := m.RenderCallFooter(); cf != "" {
			return cf
		}
	}
	parts := make([]string, 0, len(FooterKeys))
	for _, b := range FooterKeys {
		s := fmt.Sprintf("%s %s", b.Key, b.Label)
		if b.Greyed {
			s = greyedKeyStyle.Render(s)
		} else {
			s = liveKeyStyle.Render(s)
		}
		parts = append(parts, s)
	}
	footer := strings.Join(parts, "  ")
	if m.ackText != "" {
		footer += "    " + cursorMarkStyle.Render("["+m.ackText+"]")
	} else if m.ackKey != "" {
		footer += "    " + cursorMarkStyle.Render(fmt.Sprintf("[ack: %s]", m.ackKey))
	}
	return footer
}
