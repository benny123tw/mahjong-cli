package play

// KeyBinding pairs a human-readable key label with the action it triggers.
// Greyed=true marks bindings that are accepted as input but have no real
// effect in this change (no game state to mutate). The action footer renders
// greyed bindings with reduced styling so the player can see what's bound
// without being misled into thinking it works.
type KeyBinding struct {
	Key    string
	Label  string
	Greyed bool
}

// FooterKeys is the keymap rendered in the action footer, in display order.
// Movement and Quit are live; everything else is bound but inert until the
// game-loop change adds real handlers.
var FooterKeys = []KeyBinding{
	{Key: "←/→", Label: "Move"},
	{Key: "1-9", Label: "Jump"},
	{Key: "D", Label: "Discard", Greyed: true},
	{Key: "R", Label: "Riichi", Greyed: true},
	{Key: "T", Label: "Tsumo", Greyed: true},
	{Key: "P", Label: "Pon", Greyed: true},
	{Key: "C", Label: "Chi", Greyed: true},
	{Key: "K", Label: "Kan", Greyed: true},
	{Key: "Spc", Label: "Pass", Greyed: true},
	{Key: "?", Label: "Peek", Greyed: true},
	{Key: "Q", Label: "Quit"},
}
