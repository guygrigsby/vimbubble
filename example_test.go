package vimbubble_test

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/guygrigsby/vimbubble"
)

// Example shows the typical wiring inside a Bubble Tea Update loop:
// route key events through the Modal first, fall back to the textarea
// for anything it doesn't consume.
func Example() {
	ta := textarea.New()
	vim := vimbubble.New(&ta)
	vim.SetEnabled(true) // off by default; opt in.

	update := func(msg tea.KeyMsg) tea.Cmd {
		if consumed, cmds := vim.Update(msg); consumed {
			return tea.Batch(cmds...)
		}
		var cmd tea.Cmd
		ta, cmd = ta.Update(msg)
		return cmd
	}

	_ = update
}

// ExampleNew shows that a Modal starts disabled. Every key flows
// through to the textarea untouched until SetEnabled(true).
func ExampleNew() {
	ta := textarea.New()
	vim := vimbubble.New(&ta)

	// Disabled by default: vim.Update never consumes, so the host
	// textarea handles every key.
	consumed, _ := vim.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	_ = consumed // always false until SetEnabled(true).
}

// ExampleModal_SetEnabled toggles vim mode on and off, e.g. backing
// a /vim slash command in the host app. Enabling lands the composer
// in NORMAL; disabling clears any pending chord/operator state.
func ExampleModal_SetEnabled() {
	ta := textarea.New()
	vim := vimbubble.New(&ta)

	vim.SetEnabled(true)  // enters NORMAL.
	vim.SetEnabled(false) // back to pass-through.
}

// ExampleModal_Update demonstrates the (consumed, cmds) contract.
// When consumed is true, the caller must NOT forward the key to its
// textarea — vim has already handled it (and possibly synthesised
// key events into the textarea on the caller's behalf).
func ExampleModal_Update() {
	ta := textarea.New()
	vim := vimbubble.New(&ta)
	vim.SetEnabled(true)

	// In NORMAL mode, 'h' translates to a KeyLeft inside vim.Update
	// and the textarea's cursor moves. consumed=true so we stop here.
	consumed, cmds := vim.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if consumed {
		// Run any commands vim emitted (rare today).
		_ = tea.Batch(cmds...)
		return
	}
	// Not consumed: hand the key to the textarea as usual.
	ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	_ = ta
}
