// Package vimbubble adds vim-style modal editing to a
// charmbracelet/bubbles textarea.
//
// The package is built to drop in alongside any Bubble Tea app
// already using bubbles/textarea for input. You keep your textarea;
// vimbubble adds a NORMAL/INSERT mode wrapper around it.
//
// # Quick start
//
//	import (
//	    "github.com/charmbracelet/bubbles/textarea"
//	    tea "github.com/charmbracelet/bubbletea"
//	    "github.com/guygrigsby/vimbubble"
//	)
//
//	type model struct {
//	    ta  textarea.Model
//	    vim *vimbubble.Modal
//	}
//
//	func initialModel() model {
//	    ta := textarea.New()
//	    m := model{ta: ta}
//	    m.vim = vimbubble.New(&m.ta)   // disabled by default
//	    m.vim.SetEnabled(true)         // turn on; starts in NORMAL
//	    return m
//	}
//
//	func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
//	    if k, ok := msg.(tea.KeyMsg); ok {
//	        // Modal eats the key when it has a binding for it; otherwise
//	        // the textarea handles it normally.
//	        if consumed, cmds := m.vim.Update(k); consumed {
//	            return m, tea.Batch(cmds...)
//	        }
//	    }
//	    var cmd tea.Cmd
//	    m.ta, cmd = m.ta.Update(msg)
//	    return m, cmd
//	}
//
//	func (m model) View() string {
//	    return m.vim.ModeLabel() + "\n" + m.ta.View()
//	}
//
// # Supported commands
//
// NORMAL mode (text in the composer is not modified by ordinary
// keys — only the bindings below act):
//
//	h / l       cursor left / right
//	j / k       cursor down / up
//	w / b       word forward / backward
//	0 / $       line start / end
//	gg / G      buffer top / bottom
//	x           delete character forward
//	D           delete to end of line
//	~           toggle case of char under cursor + advance
//	r<x>        replace char under cursor with <x>
//	dd          delete whole line
//	dw          delete word forward
//	d$          delete to end of line
//	d0          delete to start of line
//	cw          change word (delete + INSERT)
//	c$          change to end of line (delete + INSERT)
//	c0          change to start of line (delete + INSERT)
//	cc          empty the line + INSERT
//	ciw         change inside word — works from any column within
//	            the word + INSERT
//	caw         change a word + its surrounding whitespace + INSERT
//	diw         delete inside word (stays NORMAL)
//	daw         delete a word + its surrounding whitespace
//	i / a       INSERT at / after cursor
//	I / A       INSERT at line start / end
//	o / O       open new line below / above + INSERT
//	/  :        switch to INSERT and insert the literal character
//	              (lets host apps reach a slash-command palette
//	              without first pressing `i`)
//
// INSERT mode passes every key through to the textarea, except
// Esc which returns to NORMAL.
//
// # What's not (yet) implemented
//
// Visual mode, registers, the undo stack, search (/<query>),
// jump list, find-character motions (f / t / F / T), text objects
// (iw / aw / it / ...). Open an issue or PR.
package vimbubble
