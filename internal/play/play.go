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
	"github.com/benny123tw/mahjong-cli/internal/riichi/calc"
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

// Per-zone column budgets in the four-quadrant mid-row layout. Sum
// equals targetWidth: 20 + 36 + 20 = 76 cols of content + lipgloss
// internal padding fills the rest. Used by the opponent meld renderer
// to decide between one-line, wrap, and truncate-with-+K-more layouts.
const (
	kamichaZoneWidth  = 20
	centreZoneWidth   = 36
	shimochaZoneWidth = 20
	toimenZoneWidth   = targetWidth
)

// peekUnknown is the sentinel `PeekShanten()` returns before the player has
// pressed `?` — the shanten cache is unpopulated.
const peekUnknown = -99

// HumanSeat is the seat the human player occupies. Hard-coded to South for
// v1; future changes can let users pick.
const HumanSeat = game.SeatSouth

// Model holds play-screen state. When a *game.Game pointer is supplied via
// NewWithGame, hand and pond data flow from there; the legacy New(renderer)
// path keeps the static fixture so old callers continue to render. When a
// *game.Match is supplied via NewWithMatch, the per-hand *Game is read
// through the match (and replaced after each end-of-hand transition).
type Model struct {
	match    *game.Match // nil when running against a single Game directly
	game     *game.Game  // nil for legacy fixture mode
	hand     hand.Hand   // fixture fallback
	cursor   int
	width    int
	height   int
	renderer Renderer

	// peek cache for `?` key. Zero values mean "not yet asked".
	peekShanten int
	peekMachi   []uint8
	peekHandLen int

	// peekVisible is the TUI-only visibility flag toggled by the `?` key.
	// When true, the action footer renders an extra "Wait: ..." line below
	// the keys row. Auto-cleared at every peek-cache reset site so it
	// hides on state change without requiring a second `?` press.
	peekVisible bool

	// pendingTransition holds the just-applied AdvanceFromOutcome result
	// when the player is reviewing the end-of-hand summary. While non-nil,
	// View renders an ack panel in place of the normal play layout, and
	// Update treats any keypress as "advance to next hand".
	pendingTransition *game.TransitionResult

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

// NewWithMatch constructs a Model bound to a hanchan match. The active
// per-hand *Game is read through the match and is replaced after each
// end-of-hand transition. This is the canonical entry point for `mahjong play`.
func NewWithMatch(renderer Renderer, m *game.Match) Model {
	return Model{
		match:       m,
		game:        m.CurrentGame(),
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
		// Drop bot ticks while reviewing the ack panel. The keypress on the
		// panel will resume the loop.
		if m.pendingTransition != nil {
			return m, nil
		}
		m = m.maybeAdvanceMatch()
		if m.pendingTransition != nil {
			return m, nil
		}
		return m.handleBotTick()
	case tea.KeyPressMsg:
		// Standings screen: only quit keys are honored.
		if m.match != nil && m.match.IsFinished() {
			if k := msg.String(); k == "q" || k == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}
		// Ack panel: any keypress advances to the next hand.
		if m.pendingTransition != nil {
			m.pendingTransition = nil
			if m.match != nil {
				m.game = m.match.CurrentGame()
			}
			m.peekShanten = peekUnknown
			m.peekMachi = nil
			m.peekVisible = false
			m = m.autoDrawHuman()
			return m, m.maybeBotTickCmd()
		}
		m = m.maybeAdvanceMatch()
		if m.pendingTransition != nil {
			return m, nil
		}
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

// maybeAdvanceMatch checks whether the active hand has terminated and, if
// so, applies the outcome to the bound Match. The resulting TransitionResult
// is stored as a pending acknowledgement so the TUI can render the
// end-of-hand summary before the next hand starts.
func (m Model) maybeAdvanceMatch() Model {
	if m.match == nil || m.game == nil || m.pendingTransition != nil {
		return m
	}
	if m.match.IsFinished() {
		return m
	}
	st, ok := m.game.State().(game.StateRoundOver)
	if !ok {
		return m
	}
	tr, err := m.match.AdvanceFromOutcome(st.Outcome)
	if err != nil {
		return m
	}
	m.pendingTransition = &tr
	return m
}

// autoDrawHuman fires a Draw input automatically when state lands on
// `AwaitingDraw{HumanSeat}` — the human never has to press a key to draw.
// Standard riichi UX: drawing is automatic, only the discard is a decision.
// On a successful draw the cursor jumps to the drawn tile (last index)
// so the player can immediately tsumogiri-discard without arrowing over.
func (m Model) autoDrawHuman() Model {
	if m.game == nil {
		return m
	}
	if s, ok := m.game.State().(game.StateAwaitingDraw); ok && s.Player == HumanSeat {
		if _, err := m.game.Step(game.InputDraw{}); err == nil {
			if n := len(m.game.Hand(HumanSeat)); n > 0 {
				m.cursor = n - 1
			}
		}
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
	case game.StateAwaitingChankan:
		// When the human is the shouminkan declarer, only bots can ron
		// (the declarer can't ron on their own upgrade tile). When a bot
		// is the declarer, defer to the human only when their hand could
		// produce a chankan ron.
		if s.Declarer == HumanSeat {
			return true
		}
		humanHand := m.game.Hand(HumanSeat)
		concealed := append([]tile.Tile{}, humanHand...)
		concealed = append(concealed, s.UpgradeTile)
		if hand.IsWinning(hand.Hand{Concealed: concealed}) && !m.game.IsFuriten(HumanSeat) {
			return false
		}
		return true
	case game.StateAwaitingClaims:
		// In claims state, defer to the human only when they have a legal
		// pon or chi to consider. Otherwise auto-tick: bots auto-pass in v1
		// (real bot pon/chi/ron logic ships in add-smart-ai), and there's
		// nothing for the human to decide.
		//
		// You cannot claim your own discard — when the human is the
		// discarder, no claim prompt should fire.
		if s.Discarder == HumanSeat {
			return true
		}
		humanHand := m.game.Hand(HumanSeat)
		if game.CanPon(humanHand, s.Discard) {
			return false
		}
		if len(game.CanChi(humanHand, s.Discard, s.Discarder, HumanSeat)) > 0 {
			return false
		}
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
	case game.StateAwaitingDiscard:
		m.dispatchBotDiscard(s.Player)
	case game.StateAwaitingClaims:
		m.dispatchBotClaims(s)
	case game.StateAwaitingChankan:
		m.dispatchBotChankan(s)
	}
	m = m.autoDrawHuman()
	return m, m.maybeBotTickCmd()
}

// dispatchBotChankan evaluates ron for each non-declarer bot during a
// chankan window. Pon/chi are not legal here. Each bot considers whether
// the upgrade tile completes a yaku-bearing winning shape on their hand;
// the chankan flag flips on automatically inside `Game.contextForWin`.
func (m Model) dispatchBotChankan(cs game.StateAwaitingChankan) {
	claims := map[game.Seat]game.Claim{}
	for _, seat := range []game.Seat{game.SeatEast, game.SeatSouth, game.SeatWest, game.SeatNorth} {
		if seat == cs.Declarer || seat == HumanSeat {
			continue
		}
		hd := m.game.Hand(seat)
		concealed := append([]tile.Tile{}, hd...)
		concealed = append(concealed, cs.UpgradeTile)
		// Augment the bot context so the chankan flag is observed during
		// preflight evaluation. The engine's contextForWin will set it
		// definitively when the round ends.
		ctx := m.botContextForWin(seat)
		ctx.Chankan = true
		if calc.Analyze(hand.Hand{
			Concealed: concealed,
			Winning:   cs.UpgradeTile,
			IsTsumo:   false,
			Open:      m.game.IsHandOpen(seat),
		}, ctx) != nil && !m.game.IsFuriten(seat) {
			claims[seat] = game.Claim{Kind: game.ClaimRon}
		}
	}
	_, _ = m.game.Step(game.InputResolveClaims{Claims: claims})
}

// dispatchBotDiscard runs the bot's discard-phase decisions in priority
// order: tsumo if the 14-tile hand wins, riichi if tenpai-after-discard,
// otherwise the isolation-heuristic discard. The order matches the spec'd
// Bot Decision Strategy: Tsumo > Riichi > regular discard.
func (m Model) dispatchBotDiscard(seat game.Seat) {
	hd := m.game.Hand(seat)
	// Tsumo check: build a 14-tile winning Hand and run calc.Analyze. The
	// engine's InputDeclareTsumo also re-validates, so this is just a
	// pre-flight to know whether to declare or fall through.
	if len(hd) == 14 {
		drawn := hd[len(hd)-1]
		ctx := m.botContextForWin(seat)
		if calc.Analyze(hand.Hand{
			Concealed: hd,
			Winning:   drawn,
			IsTsumo:   true,
			Open:      m.game.IsHandOpen(seat),
		}, ctx) != nil {
			if _, err := m.game.Step(game.InputDeclareTsumo{}); err == nil {
				return
			}
			// Fall through if engine rejected (e.g., yakuless edge case).
		}
	}

	bot := game.Bot{Seat: seat, Rng: m.game.Wall().Rand()}
	// Riichi check: ShouldRiichi returns the first tenpai-leaving index.
	if declare, idx := bot.ShouldRiichi(
		hd,
		m.game.Score(seat),
		m.game.Wall().LiveRemaining(),
		m.game.IsHandOpen(seat),
	); declare {
		_, _ = m.game.Step(game.InputDiscard{Index: idx, Riichi: true})
		return
	}

	danger := m.assembleDangerMap(seat, hd)
	// Fold mode: when at least one opponent has declared riichi (danger map
	// non-empty) AND the bot's 14-tile hand is shanten >= 2, switch from the
	// push-mode K=2000 blend to fold-mode K=1_000_000 so the bot picks the
	// safest tile regardless of isolation. Per the Bot Decision Strategy
	// fold-mode rule.
	var idx int
	if len(danger) > 0 && hand.Shanten(hand.Hand{Concealed: hd}) >= 2 {
		idx = max(bot.FoldDiscard(hd, danger), 0)
	} else {
		idx = max(bot.DangerAwarePickDiscard(hd, danger), 0)
	}
	_, _ = m.game.Step(game.InputDiscard{Index: idx})
}

// assembleDangerMap builds the per-tile-ID danger scores against every
// riichi-declared opponent of `seat`. Genbutsu (tile-ID in opponent's pond)
// scores 0; suji-safe (per the rank pair table — see game.SujiSafe) scores
// 1; tiles absent from the map default to 2 inside DangerAwarePickDiscard.
// Multi-riichi: min across all declarers — the safest classification wins.
//
// Returns an empty map (not nil) when no opponent is in riichi; the bot's
// PickDiscard fallback fires for the empty case in DangerAwarePickDiscard.
func (m Model) assembleDangerMap(seat game.Seat, hand []tile.Tile) map[uint8]int {
	danger := map[uint8]int{}
	setDanger := func(id uint8, level int) {
		if cur, ok := danger[id]; !ok || level < cur {
			danger[id] = level
		}
	}
	for _, opp := range []game.Seat{game.SeatEast, game.SeatSouth, game.SeatWest, game.SeatNorth} {
		if opp == seat {
			continue
		}
		if !m.game.IsRiichiDeclared(opp) {
			continue
		}
		oppPond := m.game.Discards(opp)
		for _, t := range oppPond {
			setDanger(t.ID, 0)
		}
		for _, c := range hand {
			if game.SujiSafe(oppPond, c) {
				setDanger(c.ID, 1)
			}
		}
	}
	return danger
}

// dispatchBotClaims iterates non-discarder bot seats and collects each
// bot's claim (ron > pon > chi) into a single InputResolveClaims call.
// The human's claim is NOT auto-submitted here — they drive their own
// keypress (R/P/C/Space). The engine's ResolveClaims enforces priority
// across all collected claims.
func (m Model) dispatchBotClaims(cs game.StateAwaitingClaims) {
	claims := map[game.Seat]game.Claim{}
	for _, seat := range []game.Seat{game.SeatEast, game.SeatSouth, game.SeatWest, game.SeatNorth} {
		if seat == cs.Discarder || seat == HumanSeat {
			continue
		}
		hd := m.game.Hand(seat)
		bot := game.Bot{Seat: seat, Rng: m.game.Wall().Rand()}

		// Ron: calc.Analyze on concealed+discard non-nil AND not furiten.
		concealed := append([]tile.Tile{}, hd...)
		concealed = append(concealed, cs.Discard)
		if calc.Analyze(hand.Hand{
			Concealed: concealed,
			Winning:   cs.Discard,
			IsTsumo:   false,
			Open:      m.game.IsHandOpen(seat),
		}, m.botContextForWin(seat)) != nil && !m.game.IsFuriten(seat) {
			claims[seat] = game.Claim{Kind: game.ClaimRon}
			continue
		}

		// Pon: ShouldPon checks CanPon internally; we feed it the yakuhai
		// flag and the bot's current shanten.
		isYakuhai := game.IsYakuhai(cs.Discard.ID, m.game.RoundWind(), m.game.SeatWindFor(seat))
		shanten := hand.Shanten(hand.Hand{Concealed: hd})
		if bot.ShouldPon(hd, cs.Discard, isYakuhai, shanten) {
			claims[seat] = game.Claim{Kind: game.ClaimPon}
			continue
		}

		// Chi: kamicha-only. ShouldChi enforces the kamicha rule via
		// CanChi internally, but checking here keeps the iteration cheap.
		if seat == cs.Discarder.Next() {
			if option, ok := bot.ShouldChi(hd, cs.Discard, cs.Discarder); ok {
				claims[seat] = game.Claim{Kind: game.ClaimChi, ChiTiles: option}
				continue
			}
		}
	}
	_, _ = m.game.Step(game.InputResolveClaims{Claims: claims})
}

// botContextForWin builds a calc.Context for a bot win evaluation. Mirrors
// Game.contextForWin but for a non-human seat without exposing the engine's
// full per-seat riichi state — tsumo/ron pre-flights only need the basic
// scoring context (calc.Analyze rejects yakuless wins regardless). Seat
// wind is read dealer-relative via Game.SeatWindFor so per-hand rotations
// produce the correct yakuhai/seat-wind context.
func (m Model) botContextForWin(seat game.Seat) calc.Context {
	return calc.Context{
		SeatWind:  m.game.SeatWindFor(seat),
		RoundWind: m.game.RoundWind(),
		Dora:      m.game.DoraIndicators(),
	}
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
		return m.handleRiichiOrRon(), nil
	case "p":
		return m.handlePon(), nil
	case "c":
		return m.handleChi(), nil
	case "k":
		return m.handleKan(), nil
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
	m.peekVisible = !m.peekVisible
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
	m.peekVisible = false
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

// handleRiichiOrRon dispatches the `r` key based on the current game state.
// In AwaitingDiscard{Human}, R declares riichi. In AwaitingClaims with the
// human as a non-discarder, R declares ron. Anywhere else, R is a no-op.
func (m Model) handleRiichiOrRon() Model {
	if m.game == nil {
		return m
	}
	switch s := m.game.State().(type) {
	case game.StateAwaitingDiscard:
		if s.Player == HumanSeat {
			return m.handleRiichi()
		}
	case game.StateAwaitingClaims:
		if s.Discarder != HumanSeat {
			return m.handleRon()
		}
	}
	return m
}

// handleRiichi submits InputDiscard{Index: cursor, Riichi: true}. On
// ErrIllegalRiichi, probe each precondition to surface a specific reason
// in ackText. On success, clear ackText.
func (m Model) handleRiichi() Model {
	if m.game == nil {
		return m
	}
	if _, err := m.game.Step(game.InputDiscard{Index: m.cursor, Riichi: true}); err != nil {
		m.ackText = riichiRejectionReason(m.game, m.cursor)
		return m
	}
	m.ackText = ""
	return m
}

// riichiRejectionReason explains why a riichi declaration was rejected by
// re-checking each of the four preconditions in order. Used to give the
// player a specific footer hint instead of a bare "illegal" message.
func riichiRejectionReason(g *game.Game, cursor int) string {
	if g.IsHandOpen(HumanSeat) {
		return "riichi: hand is open"
	}
	if g.Wall().LiveRemaining() < 4 {
		return "riichi: wall has <4 tiles"
	}
	humanHand := g.Hand(HumanSeat)
	if cursor < 0 || cursor >= len(humanHand) {
		return "riichi: cursor out of range"
	}
	postDiscard := append([]tile.Tile{}, humanHand[:cursor]...)
	postDiscard = append(postDiscard, humanHand[cursor+1:]...)
	if hand.Shanten(hand.Hand{Concealed: postDiscard}) != 0 {
		return "riichi: hand not tenpai"
	}
	// Score check ordering matches engine; if all other checks pass but the
	// engine still rejected, score must be the issue.
	return "riichi: insufficient funds"
}

// handleRon submits InputResolveClaims with ClaimRon. On no-yaku or furiten,
// surface a specific reason in ackText.
func (m Model) handleRon() Model {
	if m.game == nil {
		return m
	}
	cs, ok := m.game.State().(game.StateAwaitingClaims)
	if !ok {
		return m
	}
	humanHand := m.game.Hand(HumanSeat)
	concealed := append([]tile.Tile{}, humanHand...)
	concealed = append(concealed, cs.Discard)
	h := hand.Hand{
		Concealed: concealed,
		Winning:   cs.Discard,
		IsTsumo:   false,
		Open:      m.game.IsHandOpen(HumanSeat),
	}
	if calc.Analyze(h, calc.Context{
		SeatWind:  m.game.SeatWindFor(HumanSeat),
		RoundWind: m.game.RoundWind(),
		Dora:      m.game.DoraIndicators(),
	}) == nil {
		m.ackText = "ron: no yaku"
		return m
	}
	if m.game.IsFuriten(HumanSeat) {
		m.ackText = "ron: furiten"
		return m
	}
	if _, err := m.game.Step(game.InputResolveClaims{Claims: map[game.Seat]game.Claim{
		HumanSeat: {Kind: game.ClaimRon},
	}}); err != nil {
		m.ackText = "ron: " + err.Error()
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
	// You cannot claim your own discard — suppress the call-window prompt
	// entirely when the human is the discarder.
	if cs.Discarder == HumanSeat {
		return ""
	}
	humanHand := m.game.Hand(HumanSeat)
	canPon := game.CanPon(humanHand, cs.Discard)
	canChi := len(game.CanChi(humanHand, cs.Discard, cs.Discarder, HumanSeat)) > 0
	canKan := game.CanKan(humanHand, cs.Discard)

	// Compute ron legality: the hand must form a yaku-bearing winning shape
	// on `concealed + discard`, and the human must NOT be in permanent
	// furiten. The furiten case gets a `(furiten)` suffix so the player
	// understands why ron is unavailable; a no-yaku case just greys out.
	concealedPlusDiscard := append([]tile.Tile{}, humanHand...)
	concealedPlusDiscard = append(concealedPlusDiscard, cs.Discard)
	canWin := calc.Analyze(hand.Hand{
		Concealed: concealedPlusDiscard,
		Winning:   cs.Discard,
		IsTsumo:   false,
		Open:      m.game.IsHandOpen(HumanSeat),
	}, calc.Context{
		SeatWind:  m.game.SeatWindFor(HumanSeat),
		RoundWind: m.game.RoundWind(),
		Dora:      m.game.DoraIndicators(),
	}) != nil
	furiten := m.game.IsFuriten(HumanSeat)
	canRon := canWin && !furiten
	furitenBlock := canWin && furiten

	render := func(label string, active bool) string {
		if active {
			return liveKeyStyle.Render(label)
		}
		return greyedKeyStyle.Render(label)
	}
	ronLabel := "[R]on"
	if furitenBlock {
		ronLabel = "[R]on (furiten)"
	}
	parts := []string{
		render("[P]on", canPon),
		render("[C]hi", canChi),
		render("[K]an", canKan),
		render(ronLabel, canRon),
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

	var body string
	switch {
	case m.match != nil && m.match.IsFinished():
		body = m.renderStandings()
	case m.pendingTransition != nil:
		body = m.renderTransitionAck()
	default:
		body = m.renderLayout()
	}

	if m.width >= targetWidth && m.height >= targetHeight {
		body = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
	}
	return tea.NewView(body)
}

// scoreSuffix returns " · <score>" when a Match is bound, otherwise empty.
func (m Model) scoreSuffix(s game.Seat) string {
	if m.match == nil {
		return ""
	}
	return fmt.Sprintf(" · %d", m.match.Scores()[s])
}

// renderTransitionAck draws the end-of-hand reveal panel. For supported
// outcome variants (ron, tsumo, ryuukyoku) it delegates to the enriched
// `renderEndPanel` (four-row reveal + yaku/han/fu/deltas for wins;
// four-row reveal + tenpai/noten labels + deltas for ryuukyoku). For
// unsupported variants it falls back to the minimal outcome-line +
// deltas + renchan-or-next-hand summary.
func (m Model) renderTransitionAck() string {
	tr := m.pendingTransition
	if tr == nil {
		return ""
	}
	if rich := m.renderEndPanel(); rich != "" {
		return rich
	}
	header := "Hand complete"
	if m.game != nil {
		if st, ok := m.game.State().(game.StateRoundOver); ok {
			header = describeOutcome(st.Outcome)
		}
	}
	rows := []string{
		statusStyle.Render(header),
		"",
		labelStyle.Render(formatDeltasRow(tr.Deltas, tr.NewTotals)),
		"",
	}
	if tr.Renchan {
		rows = append(rows, labelStyle.Render(fmt.Sprintf("Renchan — Honba %d", tr.NewHonba)))
	} else if tr.MatchOutcome == nil {
		rows = append(rows, labelStyle.Render(fmt.Sprintf("Next hand: index %d", tr.NewHandIndex)))
	}
	rows = append(rows, "", liveKeyStyle.Render(endPanelFooter))
	return strings.Join(rows, "\n")
}

// renderStandings draws the end-of-match standings: four rows sorted by
// score descending, the match-end reason, and a quit prompt.
func (m Model) renderStandings() string {
	if m.match == nil {
		return ""
	}
	scores := m.match.Scores()
	type entry struct {
		seat   game.Seat
		points int
	}
	rows := []entry{}
	for s := range game.Seat(4) {
		rows = append(rows, entry{seat: s, points: scores[s]})
	}
	for i := range rows {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].points > rows[i].points {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
	out := []string{statusStyle.Render("Hanchan complete")}
	for _, r := range rows {
		out = append(out, labelStyle.Render(fmt.Sprintf("  %s  %6d", seatLabel(r.seat), r.points)))
	}
	out = append(out, "")
	if oc := m.match.FinalOutcome(); oc != nil {
		reason := oc.Reason
		if oc.Reason == "tobi" {
			reason = "tobi: " + seatLabel(oc.BustSeat)
		}
		out = append(out, labelStyle.Render("Reason: "+reason))
	}
	out = append(out, "", liveKeyStyle.Render("[q] Quit"))
	return strings.Join(out, "\n")
}

func describeOutcome(o game.Outcome) string {
	switch v := o.(type) {
	case game.OutcomeRon:
		amount := 0
		if v.Result != nil {
			amount = v.Result.Award.Total
		}
		return fmt.Sprintf("%s ron from %s — %d", seatLabel(v.Winner), seatLabel(v.Loser), amount)
	case game.OutcomeTsumo:
		amount := 0
		if v.Result != nil {
			amount = v.Result.Award.Total
		}
		return fmt.Sprintf("%s tsumo — %d", seatLabel(v.Winner), amount)
	case game.OutcomeRyuukyoku:
		if len(v.TenpaiPlayers) == 0 {
			return "Ryuukyoku — no tenpai"
		}
		names := make([]string, 0, len(v.TenpaiPlayers))
		for _, s := range v.TenpaiPlayers {
			names = append(names, seatLabel(s))
		}
		return "Ryuukyoku — tenpai: " + strings.Join(names, ", ")
	}
	return "Hand complete"
}

func formatDeltasRow(deltas, totals [4]int) string {
	parts := make([]string, 0, 4)
	for s := range game.Seat(4) {
		sign := "+"
		if deltas[s] < 0 {
			sign = ""
		}
		parts = append(
			parts,
			fmt.Sprintf("%s %s%d (→%d)", seatLabel(s), sign, deltas[s], totals[s]),
		)
	}
	return strings.Join(parts, "   ")
}

func seatLabel(s game.Seat) string {
	switch s {
	case game.SeatEast:
		return "East"
	case game.SeatSouth:
		return "South"
	case game.SeatWest:
		return "West"
	case game.SeatNorth:
		return "North"
	}
	return "?"
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
	statusStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Bold(true)
	labelStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	cursorMarkStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	focusedTileStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	greyedKeyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	liveKeyStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	furitenBadgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
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
	handLabel := "East 1"
	honba := 0
	sticks := 0
	humanScore := 25000
	if m.match != nil {
		handLabel = m.match.HandLabel()
		honba = m.match.Honba()
		sticks = m.match.RiichiSticks()
		humanScore = m.match.Scores()[HumanSeat]
	}
	return statusStyle.Render(
		fmt.Sprintf(
			"%s  ·  Honba %d  ·  Riichi %d  ·  Wall %d  ·  Dora: %s  ·  Seat: South  ·  Score %d",
			handLabel,
			honba,
			sticks,
			wallRemaining,
			m.firstDoraIndicator(),
			humanScore,
		),
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
	score := 25000
	if m.match != nil {
		score = m.match.Scores()[game.SeatNorth]
	}
	label := labelStyle.Render(fmt.Sprintf("        Toimen — North · %d", score))
	melds := m.renderOpponentMelds(game.SeatNorth, toimenZoneWidth)
	row := m.renderBackRow(13)
	pond := renderPondZone(m.Pond(game.SeatNorth), m.renderer)
	parts := []string{label}
	if melds != "" {
		parts = append(parts, melds)
	}
	parts = append(parts, row)
	if pond != "" {
		parts = append(parts, pond)
	}
	return strings.Join(parts, "\n")
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

	kamiStyled := lipgloss.NewStyle().Width(kamichaZoneWidth).Render(kami)
	centreStyled := lipgloss.NewStyle().Width(centreZoneWidth).Render(centre)
	shimoStyled := lipgloss.NewStyle().Width(shimochaZoneWidth).Render(shimo)
	return lipgloss.JoinHorizontal(lipgloss.Top, kamiStyled, centreStyled, shimoStyled)
}

func (m Model) renderKamichaColumn() string {
	label := labelStyle.Render("Kamicha · East" + m.scoreSuffix(game.SeatEast))
	melds := m.renderOpponentMelds(game.SeatEast, kamichaZoneWidth)
	pond := renderPondZone(m.Pond(game.SeatEast), m.renderer)
	parts := []string{label}
	if melds != "" {
		parts = append(parts, melds)
	}
	if pond != "" {
		parts = append(parts, pond)
	}
	return strings.Join(parts, "\n")
}

func (m Model) renderShimochaColumn() string {
	label := labelStyle.Render("Shimocha · West" + m.scoreSuffix(game.SeatWest))
	melds := m.renderOpponentMelds(game.SeatWest, shimochaZoneWidth)
	pond := renderPondZone(m.Pond(game.SeatWest), m.renderer)
	parts := []string{label}
	if melds != "" {
		parts = append(parts, melds)
	}
	if pond != "" {
		parts = append(parts, pond)
	}
	return strings.Join(parts, "\n")
}

func (m Model) renderCentreInfo() string {
	wall := 70
	if m.game != nil {
		wall = m.game.Wall().LiveRemaining()
	}
	handLabel := "East 1"
	honba := 0
	humanScore := 25000
	if m.match != nil {
		handLabel = m.match.HandLabel()
		honba = m.match.Honba()
		humanScore = m.match.Scores()[HumanSeat]
	}
	return labelStyle.Render(fmt.Sprintf(
		"     Round: %s\n     Honba:  %d\n     Wall:   %d\n     Dora:   %s\n     You:    South · %d",
		handLabel,
		honba,
		wall,
		m.firstDoraIndicator(),
		humanScore,
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
	cells := make([]string, 0, len(tiles)+1)
	for i, t := range tiles {
		tileLines := m.renderer.Tile(t)
		tileBlock := strings.Join(tileLines, "\n")
		if i == m.cursor {
			tileBlock = focusedTileStyle.Render(tileBlock)
		}
		// In AwaitingDiscard{Human} with 14 tiles, the just-drawn 14th tile
		// lives at index 13 and SHALL be visually separated from the sorted
		// main hand by a one-tile-slot gap (game-loop spec, Human Hand
		// Canonical Sort Invariant). The gap is rendered as Width() spaces
		// per row, matching the renderer's tile cell count.
		if i == 13 && m.shouldShowDrawnTileGap(len(tiles)) {
			cells = append(cells, m.handGap())
		}
		cells = append(cells, tileBlock)
	}
	concealed := lipgloss.JoinHorizontal(lipgloss.Top, cells...)
	melds := m.renderOpenMeldsForSeat(HumanSeat)
	if melds == "" {
		return concealed
	}
	// Layout: 2-tile-width gap between concealed and melds. When the combined
	// width exceeds 80, stack melds onto a second line below the concealed row.
	gap := m.handGap() + m.handGap()
	combined := lipgloss.JoinHorizontal(lipgloss.Top, concealed, gap, melds)
	if lipgloss.Width(combined) > targetWidth {
		return lipgloss.JoinVertical(lipgloss.Top, concealed, melds)
	}
	return combined
}

// renderOpenMeldsForSeat renders `seat`'s called melds (pon, chi, ankan,
// minkan, shouminkan) as a horizontally-joined block. The seat-source
// marker ([E]/[S]/[W]/[N]) is attached immediately to the LEFT of the
// called tile — the tile most recently grabbed for that meld — so the
// reader can see which physical tile came from another seat. Ankan has
// no called tile, so [A] is rendered as a meld-level prefix. Returns ""
// when the seat has no open melds.
//
// Generalized from the original `renderOpenMelds()` (HumanSeat-only) to
// support the end-of-hand reveal panel where every seat's melds render.
func (m Model) renderOpenMeldsForSeat(seat game.Seat) string {
	if m.game == nil {
		return ""
	}
	melds := m.game.Melds(seat)
	if len(melds) == 0 {
		return ""
	}
	blocks := make([]string, 0, len(melds)*2)
	for i, meld := range melds {
		if i > 0 {
			blocks = append(blocks, m.handGap())
		}
		blocks = append(blocks, m.renderSingleMeld(meld))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, blocks...)
}

// renderSingleMeld renders one open meld as a horizontally-joined block:
// seat-source marker attached to the called tile (or as a meld-level
// prefix for ankan), then the meld's tiles. Used by both the multi-meld
// joiner and the per-zone wrap/truncate logic in renderOpponentMelds.
func (m Model) renderSingleMeld(meld game.Meld) string {
	marker := openMeldMarker(meld)
	calledIdx := calledTileIndex(meld)
	parts := make([]string, 0, len(meld.Tiles)+2)
	if calledIdx < 0 {
		parts = append(parts, lipgloss.NewStyle().Render(marker+" "))
	}
	for j, t := range meld.Tiles {
		if j == calledIdx {
			parts = append(parts, lipgloss.NewStyle().Render(marker))
		}
		tileLines := m.renderer.Tile(t)
		parts = append(parts, strings.Join(tileLines, "\n"))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// renderOpponentMelds renders an opponent seat's open melds inside a
// fixed-width zone. Layout cascade:
//
//  1. Empty meld list → "".
//  2. Single-line block fits in zoneWidth → return it as-is.
//  3. Block exceeds zoneWidth → wrap by greedily packing melds onto
//     up to 2 lines.
//  4. Even 2 lines do not fit all melds → truncate at the first N
//     melds that fit, then append `+K more` on a 3rd line.
//
// Used by the four-quadrant per-opponent zone renderers (Kamicha,
// Toimen, Shimocha) so live play surfaces what each bot has called.
func (m Model) renderOpponentMelds(seat game.Seat, zoneWidth int) string {
	if m.game == nil {
		return ""
	}
	melds := m.game.Melds(seat)
	if len(melds) == 0 {
		return ""
	}
	full := m.renderOpenMeldsForSeat(seat)
	if lipgloss.Width(full) <= zoneWidth {
		return full
	}
	// Greedy fit: pack melds left-to-right onto up to 2 lines, each
	// line bounded by zoneWidth. Inter-meld gap matches handGap() to
	// stay consistent with the single-line meld block.
	gap := m.handGap()
	gapWidth := lipgloss.Width(gap)
	const maxLines = 2
	lines := make([][]string, 0, maxLines)
	current := []string{}
	currentWidth := 0
	consumed := 0
	for _, meld := range melds {
		block := m.renderSingleMeld(meld)
		w := lipgloss.Width(block)
		need := w
		if len(current) > 0 {
			need += gapWidth
		}
		if currentWidth+need <= zoneWidth {
			if len(current) > 0 {
				current = append(current, gap)
			}
			current = append(current, block)
			currentWidth += need
			consumed++
			continue
		}
		// Current line is full. Start a new line if budget allows.
		if len(current) > 0 {
			lines = append(lines, current)
			current = nil
		}
		if len(lines) >= maxLines {
			break
		}
		// Single-meld line: even if it overflows zoneWidth alone, place
		// it on its own line (better than dropping). zoneWidth = 20 fits
		// any 3-tile pon (~10 cols) and any 4-tile kan (~16 cols) under
		// Unicode, so this branch is the typical wrap path.
		current = []string{block}
		currentWidth = w
		consumed++
	}
	if len(current) > 0 && len(lines) < maxLines {
		lines = append(lines, current)
	}
	rendered := make([]string, 0, len(lines)+1)
	for _, lineParts := range lines {
		rendered = append(rendered, lipgloss.JoinHorizontal(lipgloss.Top, lineParts...))
	}
	if remaining := len(melds) - consumed; remaining > 0 {
		rendered = append(rendered,
			labelStyle.Render(fmt.Sprintf("+%d more", remaining)))
	}
	return lipgloss.JoinVertical(lipgloss.Top, rendered...)
}

// openMeldMarker returns the bracketed seat-source marker for an open meld.
// Ankan returns "[A]" (no source seat); every other kind returns the From
// seat letter.
func openMeldMarker(meld game.Meld) string {
	if meld.Kind == game.MeldKan && meld.KanKind == game.KanAnkan {
		return "[A]"
	}
	switch meld.From {
	case game.SeatEast:
		return "[E]"
	case game.SeatSouth:
		return "[S]"
	case game.SeatWest:
		return "[W]"
	case game.SeatNorth:
		return "[N]"
	}
	return "[?]"
}

// calledTileIndex returns the index within meld.Tiles of the tile most
// recently grabbed for that meld — the one whose seat-source marker is
// attached as a pointer. Returns -1 for ankan (no called tile; marker
// renders as a meld-level prefix instead).
//
// Tile positions follow how the engine constructs each meld kind:
//   - Pon         [d, d, d]              → called at idx 0
//   - Chi         [c1, c2, d]            → called at idx 2 (last)
//   - Minkan      [d, c1, c2, c3]        → called at idx 0
//   - Shouminkan  [d, d, d, upgrade]     → newest grab at idx 3 (upgrade)
//   - Ankan       (concealed)            → -1
func calledTileIndex(meld game.Meld) int {
	switch meld.Kind {
	case game.MeldPon:
		return 0
	case game.MeldChi:
		return len(meld.Tiles) - 1
	case game.MeldKan:
		switch meld.KanKind {
		case game.KanMinkan:
			return 0
		case game.KanShouminkan:
			return len(meld.Tiles) - 1
		}
	}
	return -1
}

// shouldShowDrawnTileGap reports whether the drawn-tile gap separator should
// be inserted between indices 12 and 13 of the human's hand. The gap appears
// only when the underlying state is AwaitingDiscard{Human} AND the hand has
// 14 tiles (a draw has just landed and the player has not yet discarded).
func (m Model) shouldShowDrawnTileGap(handLen int) bool {
	if handLen != 14 || m.game == nil {
		return false
	}
	st, ok := m.game.State().(game.StateAwaitingDiscard)
	return ok && st.Player == HumanSeat
}

// handGap returns a lipgloss block of Width()×Lines() whitespace, used as
// the visual separator between the sorted main hand and the drawn 14th tile.
func (m Model) handGap() string {
	width, lines := m.renderer.Width(), m.renderer.Lines()
	row := strings.Repeat(" ", width)
	if lines <= 1 {
		return row
	}
	rows := make([]string, lines)
	for i := range rows {
		rows[i] = row
	}
	return strings.Join(rows, "\n")
}

func (m Model) renderFooter() string {
	if m.game != nil {
		if cf := m.RenderCallFooter(); cf != "" {
			return cf
		}
	}
	kanLive := humanKanLegal(m)
	parts := make([]string, 0, len(FooterKeys))
	for _, b := range FooterKeys {
		s := fmt.Sprintf("%s %s", b.Key, b.Label)
		live := !b.Greyed
		if b.Key == "K" {
			live = kanLive
		}
		if live {
			s = liveKeyStyle.Render(s)
		} else {
			s = greyedKeyStyle.Render(s)
		}
		parts = append(parts, s)
	}
	footer := strings.Join(parts, "  ")
	if m.ackText != "" {
		footer += "    " + cursorMarkStyle.Render("["+m.ackText+"]")
	} else if m.ackKey != "" {
		footer += "    " + cursorMarkStyle.Render(fmt.Sprintf("[ack: %s]", m.ackKey))
	}
	if badge := m.renderFuritenBadge(); badge != "" {
		footer += "    " + badge
	}
	if m.peekVisible {
		footer += "\n" + m.renderPeekLine()
	}
	return footer
}

// renderFuritenBadge returns the standalone furiten badge ("[FURITEN]" in
// Unicode mode, "(furiten)" in ASCII mode) when the human is at tenpai AND
// in furiten AND it's the human's own turn cycle. Returns empty string in
// every other case. The call window handles its own furiten indicator via
// the [R]on (furiten) Ron-button suffix.
func (m Model) renderFuritenBadge() string {
	if m.game == nil {
		return ""
	}
	switch s := m.game.State().(type) {
	case game.StateAwaitingDraw:
		if s.Player != HumanSeat {
			return ""
		}
	case game.StateAwaitingDiscard:
		if s.Player != HumanSeat {
			return ""
		}
	default:
		return ""
	}
	humanHand := m.game.Hand(HumanSeat)
	if hand.Shanten(hand.Hand{Concealed: humanHand}) != 0 {
		return ""
	}
	if !m.game.IsFuriten(HumanSeat) {
		return ""
	}
	if _, isASCII := m.renderer.(ASCIIRenderer); isASCII {
		return "(furiten)"
	}
	return furitenBadgeStyle.Render("[FURITEN]")
}

// renderPeekLine returns the "Wait: ..." line shown beneath the action keys
// when peekVisible is true. Tenpai hands list each machi tile ID via
// `tile.Tile.String()`; non-tenpai hands render "(not tenpai)".
func (m Model) renderPeekLine() string {
	if m.peekShanten == 0 && len(m.peekMachi) > 0 {
		ids := make([]string, 0, len(m.peekMachi))
		for _, id := range m.peekMachi {
			ids = append(ids, (tile.Tile{ID: id}).String())
		}
		return "Wait: " + strings.Join(ids, " ")
	}
	return "Wait: (not tenpai)"
}
