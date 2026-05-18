package vimbubble

import (
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// Mode is the current vim mode. The zero value is Disabled, meaning
// vimbubble passes every key through untouched.
type Mode uint8

const (
	// Disabled is the off-switch: vimbubble.Update never consumes a
	// key, so the host textarea behaves like a plain text input.
	Disabled Mode = iota
	// Normal is vim's command mode: arrow-key cursor motion + the
	// vim verbs (x, dw, cw, ~, r, …). Letter keys are swallowed
	// rather than inserted so a stray "x" doesn't paste an x into
	// the buffer.
	Normal
	// Insert is plain typing: every key passes through to the
	// textarea except Esc, which returns the user to Normal.
	Insert
)

// String returns a short human-readable label ("disabled", "normal",
// "insert"). For a styled footer label, see (*Modal).ModeLabel.
func (m Mode) String() string {
	switch m {
	case Normal:
		return "normal"
	case Insert:
		return "insert"
	}
	return "disabled"
}

// Modal wraps a textarea.Model with modal editing. Caller owns the
// textarea — Modal doesn't allocate it and never replaces it. Modal only
// reads + mutates the textarea's value + cursor.
type Modal struct {
	composer *textarea.Model
	mode     Mode

	// chord tracks the first key of a two-key sequence (gg, dd, cw,
	// d$, …) with a short timeout so a stray "g" doesn't permanently
	// trap the composer in a half-chord state.
	chord chord

	// operatorPending captures an operator that needs an argument
	// (today: `r` waiting for the replacement char). Reset on the
	// next keystroke regardless of validity so we never get stuck.
	operatorPending rune
}

// New returns a Modal attached to ta. Starts disabled.
//
// The textarea's cursor row/col are read via reflect+unsafe (Bubbles
// doesn't expose them publicly), so a future bubbles release that
// renames those fields would silently break ~ / r / cw. The
// TestComposerCursorReadable test pins the contract.
func New(ta *textarea.Model) *Modal {
	return &Modal{composer: ta}
}

// Mode returns the current mode.
func (v *Modal) Mode() Mode { return v.mode }

// SetMode forces the mode. Useful when a host app needs to bring the
// composer into Insert after running a command (e.g. dropping into
// edit-mode) or wants to drop back to Normal after a slash-command
// finishes. SetMode(Disabled) clears pending state the same way
// SetEnabled(false) does.
func (v *Modal) SetMode(m Mode) {
	v.mode = m
	if m == Disabled {
		v.chord.reset()
		v.operatorPending = 0
	}
}

// IsEnabled reports whether vim is active. Equivalent to
// v.Mode() != Disabled. Provided as a convenience for callers that
// want to gate behaviour without typing the comparison.
func (v *Modal) IsEnabled() bool { return v.mode != Disabled }

// SetEnabled flips vim mode on/off. Enabling lands the composer in
// Normal (matches the way vim itself opens). Disabling clears any
// pending chord/operator state.
func (v *Modal) SetEnabled(on bool) {
	if on {
		v.mode = Normal
		return
	}
	v.mode = Disabled
	v.chord.reset()
	v.operatorPending = 0
}

// ModeLabel returns a styled label like "-- NORMAL --" suitable for
// rendering near the composer. Empty when disabled — callers can
// switch on the empty string to decide whether to render anything.
func (v *Modal) ModeLabel() string {
	switch v.mode {
	case Normal:
		return "-- NORMAL --"
	case Insert:
		return "-- INSERT --"
	}
	return ""
}

// Update routes a key event through vim. Returns (consumed, cmds):
//   - consumed=true → vim handled this key. The caller should NOT
//     forward msg to the textarea. cmds is a slice of any commands
//     vim needs the Bubble Tea runtime to run (today: usually nil).
//   - consumed=false → vim isn't interested in this key. The caller
//     handles it as it would have without vim (e.g. textarea.Update).
//
// Disabled mode never consumes — Modal is invisible until enabled.
func (v *Modal) Update(msg tea.KeyMsg) (consumed bool, cmds []tea.Cmd) {
	switch v.mode {
	case Disabled:
		return false, nil

	case Insert:
		if msg.String() == "esc" {
			v.mode = Normal
			return true, nil
		}
		// Pass everything else through. We don't synthesise it; the
		// caller's textarea.Update will see the same key.
		return false, nil

	case Normal:
		forward, next, _ := v.handleNormal(msg)
		v.mode = next
		// Forward synthesised key events to the textarea so its
		// internal cursor / word-walk logic runs.
		for _, f := range forward {
			var c tea.Cmd
			*v.composer, c = v.composer.Update(f)
			if c != nil {
				cmds = append(cmds, c)
			}
		}
		return true, cmds
	}
	return false, nil
}

// chord tracks the first key of a two-key sequence. Cleared when the
// timeout expires so the user isn't trapped if they pressed a chord
// starter and then walked away.
// chord tracks the partial state of a multi-key vim sequence:
//   - 2-key:  operator + motion       — cw, dd, gg, d$, …
//   - 3-key:  operator + specifier + object — ciw, caw, diw, daw
//
// The specifier (`i` for "inner", `a` for "around") bridges operator
// and text-object keys. Bare `i` / `a` are mode switches in vim's
// grammar; we only read them as specifiers when an operator is
// already pending. Cleared after chordTimeout so the user isn't
// trapped after pressing a chord starter and walking off.
type chord struct {
	pending   rune // the operator: 'c', 'd', 'g'
	specifier rune // 'i' or 'a', set only mid-sequence
	startedAt time.Time
}

const chordTimeout = 800 * time.Millisecond

func (c *chord) reset() {
	c.pending = 0
	c.specifier = 0
	c.startedAt = time.Time{}
}

func (c *chord) start(r rune) {
	c.reset()
	c.pending = r
	c.startedAt = time.Now()
}

// setSpecifier advances a pending operator into a 3-key sequence by
// stamping the inner/around specifier. Refreshes the timeout so the
// user gets another chordTimeout window to land the object key.
func (c *chord) setSpecifier(r rune) {
	c.specifier = r
	c.startedAt = time.Now()
}

// stale reports whether the pending sequence has aged out.
func (c *chord) stale() bool {
	return c.pending != 0 && time.Since(c.startedAt) > chordTimeout
}
