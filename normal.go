package vimbubble

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleNormal interprets a key against the NORMAL-mode keymap.
// Returns:
//   - forward: one or more synthesised tea.KeyMsgs the textarea
//     should process. Maps `h` → KeyLeft and so on, so the
//     textarea's own cursor + word-walk logic stays the source of
//     truth for character-level positioning.
//   - nextMode: the mode the composer should be in after this key.
//   - handled: true if the key was a vim command. Always true in
//     practice — NORMAL mode swallows unrecognised keys rather than
//     letting them insert into the buffer.
func (v *Modal) handleNormal(msg tea.KeyMsg) (forward []tea.KeyMsg, nextMode Mode, handled bool) {
	s := msg.String()

	// Pending state takes priority: when `r` is waiting for its arg
	// the next rune MUST replace the char under the cursor regardless
	// of what it would otherwise mean. Same for an in-progress chord:
	// the next rune completes it, not gets re-routed through motion
	// dispatch (`w` after `c` is "change word", not "move forward").
	if v.operatorPending == 'r' {
		v.operatorPending = 0
		if len(msg.Runes) != 1 {
			return nil, Normal, true
		}
		replaceCharAtCursor(v.composer, msg.Runes[0])
		return nil, Normal, true
	}
	if v.chord.pending != 0 {
		if len(msg.Runes) != 1 {
			// Non-printable cancels the chord (Esc, arrows, etc).
			v.chord.reset()
			return nil, Normal, true
		}
		combo, _ := v.chord.consume(msg.Runes[0])
		switch combo {
		case "gg":
			return []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'<'}, Alt: true}}, Normal, true
		case "dd":
			return []tea.KeyMsg{
				{Type: tea.KeyHome},
				{Type: tea.KeyCtrlK},
				{Type: tea.KeyDelete},
			}, Normal, true
		case "dw":
			v.deleteWord()
			return nil, Normal, true
		case "d$":
			return []tea.KeyMsg{{Type: tea.KeyCtrlK}}, Normal, true
		case "d0":
			return []tea.KeyMsg{{Type: tea.KeyCtrlU}}, Normal, true
		case "cw":
			v.deleteWord()
			return nil, Insert, true
		case "c$":
			return []tea.KeyMsg{{Type: tea.KeyCtrlK}}, Insert, true
		case "c0":
			return []tea.KeyMsg{{Type: tea.KeyCtrlU}}, Insert, true
		case "cc":
			row, _ := composerCursor(v.composer)
			clearRow(v.composer, row)
			return nil, Insert, true
		}
		// Unknown combo: swallow silently. Matches vim — beeping
		// every typo would be brutal.
		return nil, Normal, true
	}

	// Mode switches first — these don't move the cursor.
	switch s {
	case "i":
		return nil, Insert, true
	case "a":
		return []tea.KeyMsg{{Type: tea.KeyRight}}, Insert, true
	case "I":
		return []tea.KeyMsg{{Type: tea.KeyHome}}, Insert, true
	case "A":
		return []tea.KeyMsg{{Type: tea.KeyEnd}}, Insert, true
	case "o":
		return []tea.KeyMsg{
			{Type: tea.KeyEnd},
			{Type: tea.KeyEnter},
		}, Insert, true
	case "O":
		return []tea.KeyMsg{
			{Type: tea.KeyHome},
			{Type: tea.KeyEnter},
			{Type: tea.KeyUp},
		}, Insert, true
	}

	// Movement.
	switch s {
	case "h":
		return []tea.KeyMsg{{Type: tea.KeyLeft}}, Normal, true
	case "l":
		return []tea.KeyMsg{{Type: tea.KeyRight}}, Normal, true
	case "j":
		return []tea.KeyMsg{{Type: tea.KeyDown}}, Normal, true
	case "k":
		return []tea.KeyMsg{{Type: tea.KeyUp}}, Normal, true
	case "0":
		return []tea.KeyMsg{{Type: tea.KeyHome}}, Normal, true
	case "$":
		return []tea.KeyMsg{{Type: tea.KeyEnd}}, Normal, true
	case "w":
		return []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'f'}, Alt: true}}, Normal, true
	case "b":
		return []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'b'}, Alt: true}}, Normal, true
	case "G":
		return []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'>'}, Alt: true}}, Normal, true
	}

	// Edits that don't need an argument.
	switch s {
	case "x":
		return []tea.KeyMsg{{Type: tea.KeyDelete}}, Normal, true
	case "D":
		return []tea.KeyMsg{{Type: tea.KeyCtrlK}}, Normal, true
	case "~":
		toggleCaseAtCursor(v.composer)
		return nil, Normal, true
	case "r":
		// Wait for the next keystroke; that key replaces the char at
		// the cursor without advancing. operatorPending is checked at
		// the top of this function on the next call.
		v.operatorPending = 'r'
		return nil, Normal, true
	}

	// `/` and `:` are vim's command-mode triggers. Host apps that
	// implement a slash-command palette want NORMAL-mode `/` to
	// drop into INSERT and pass the slash through, so users don't
	// have to first press `i` to reach their commands.
	if s == "/" || s == ":" {
		return []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'/'}}}, Insert, true
	}

	// Start a chord on the operator keys. Completion is handled at
	// the top of this function via v.chord.consume.
	if len(msg.Runes) == 1 {
		r := msg.Runes[0]
		if r == 'g' || r == 'd' || r == 'c' {
			v.chord.start(r)
			return nil, Normal, true
		}
	}

	// Unrecognised key — swallow rather than insert.
	return nil, Normal, true
}
