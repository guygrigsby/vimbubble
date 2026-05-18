package vimbubble

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// newTestModal builds a Modal attached to a wide-enough textarea for the
// keymap tests to run without their cursor falling off the right
// edge. Mode is Disabled at the start; tests enable explicitly.
func newTestModal(t *testing.T) (*Modal, *textarea.Model) {
	t.Helper()
	ta := textarea.New()
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.Focus()
	return New(&ta), &ta
}

func seedNormal(t *testing.T, v *Modal, ta *textarea.Model, text string) {
	t.Helper()
	v.SetEnabled(true)
	// Switch to Insert + type the seed.
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	for _, r := range text {
		// Insert mode passes through, so the caller's textarea.Update
		// receives the key. Mirror that here.
		consumed, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		if !consumed {
			*ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
	}
	// Esc → Normal.
	v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	// 0 → start of line.
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
}

func pressNormal(v *Modal, ta *textarea.Model, s string) {
	for _, r := range s {
		consumed, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		if !consumed {
			*ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
	}
}

func TestModeString(t *testing.T) {
	cases := []struct {
		m    Mode
		want string
	}{
		{Disabled, "disabled"},
		{Normal, "normal"},
		{Insert, "insert"},
	}
	for _, c := range cases {
		if got := c.m.String(); got != c.want {
			t.Errorf("Mode(%d).String(): got %q, want %q", c.m, got, c.want)
		}
	}
}

func TestDisabledNeverConsumes(t *testing.T) {
	v, _ := newTestModal(t)
	// Default mode is Disabled.
	consumed, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if consumed {
		t.Errorf("Disabled mode consumed a key — host textarea should handle everything")
	}
	if label := v.ModeLabel(); label != "" {
		t.Errorf("ModeLabel when disabled: got %q, want \"\"", label)
	}
}

func TestSetEnabledLandsInNormal(t *testing.T) {
	v, _ := newTestModal(t)
	v.SetEnabled(true)
	if v.Mode() != Normal {
		t.Errorf("after Enable: mode=%v, want Normal", v.Mode())
	}
	if label := v.ModeLabel(); label != "-- NORMAL --" {
		t.Errorf("ModeLabel: got %q, want %q", label, "-- NORMAL --")
	}
}

func TestInsertEscReturnsToNormal(t *testing.T) {
	v, _ := newTestModal(t)
	v.SetEnabled(true)
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}) // i → Insert
	if v.Mode() != Insert {
		t.Fatalf("i should enter Insert, got %v", v.Mode())
	}
	v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if v.Mode() != Normal {
		t.Errorf("Esc from Insert: mode=%v, want Normal", v.Mode())
	}
}

func TestInsertPassesKeysThrough(t *testing.T) {
	v, ta := newTestModal(t)
	v.SetEnabled(true)
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}) // Insert
	// Plain letters in Insert: vim doesn't consume; host typing flows
	// to the textarea unchanged.
	for _, r := range "hi" {
		consumed, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		if consumed {
			t.Errorf("Insert mode consumed %q — should pass through", r)
		}
		*ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if v := ta.Value(); v != "hi" {
		t.Errorf("textarea after typing: got %q, want hi", v)
	}
}

func TestNormalSwallowsLetters(t *testing.T) {
	v, ta := newTestModal(t)
	v.SetEnabled(true)
	// Normal mode should consume + swallow plain letters so they
	// don't insert.
	before := ta.Value()
	consumed, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if !consumed {
		t.Errorf("Normal mode let `z` through")
	}
	if ta.Value() != before {
		t.Errorf("`z` in Normal modified buffer: %q → %q", before, ta.Value())
	}
}

func TestNormalMotion_h_l(t *testing.T) {
	v, ta := newTestModal(t)
	seedNormal(t, v, ta, "abc")
	// At col 0. `l` → col 1. `l` → col 2. `h` → col 1.
	pressNormal(v, ta, "ll")
	_, col := composerCursor(ta)
	if col != 2 {
		t.Errorf("after ll: col=%d, want 2", col)
	}
	pressNormal(v, ta, "h")
	_, col = composerCursor(ta)
	if col != 1 {
		t.Errorf("after h: col=%d, want 1", col)
	}
}

func TestAppendOpensInsertAfterCursor(t *testing.T) {
	v, ta := newTestModal(t)
	seedNormal(t, v, ta, "abc")
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if v.Mode() != Insert {
		t.Fatalf("`a` should enter Insert, got %v", v.Mode())
	}
	// Type X — passes through to textarea, lands between a and b.
	for _, r := range "X" {
		consumed, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		if !consumed {
			*ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
	}
	if got := ta.Value(); got != "aXbc" {
		t.Errorf("a then X: got %q, want aXbc", got)
	}
}

func TestTilde_TogglesCaseAndAdvances(t *testing.T) {
	v, ta := newTestModal(t)
	seedNormal(t, v, ta, "Hello")
	pressNormal(v, ta, "~")
	if got := ta.Value(); got != "hello" {
		t.Errorf("first ~: %q, want hello", got)
	}
	_, col := composerCursor(ta)
	if col != 1 {
		t.Errorf("cursor after first ~: col=%d, want 1", col)
	}
}

func TestTilde_NonCasedAdvances(t *testing.T) {
	v, ta := newTestModal(t)
	seedNormal(t, v, ta, "1bc")
	pressNormal(v, ta, "~")
	if got := ta.Value(); got != "1bc" {
		t.Errorf("~ on digit changed text: %q", got)
	}
	_, col := composerCursor(ta)
	if col != 1 {
		t.Errorf("cursor after ~ on digit: col=%d, want 1", col)
	}
}

func TestReplace_ReplacesCharStaysAtCursor(t *testing.T) {
	v, ta := newTestModal(t)
	seedNormal(t, v, ta, "cat")
	pressNormal(v, ta, "rb")
	if got := ta.Value(); got != "bat" {
		t.Errorf("r b: %q, want bat", got)
	}
	_, col := composerCursor(ta)
	if col != 0 {
		t.Errorf("cursor after r: col=%d, want 0", col)
	}
}

func TestReplace_EscCancels(t *testing.T) {
	v, ta := newTestModal(t)
	seedNormal(t, v, ta, "cat")
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if got := ta.Value(); got != "cat" {
		t.Errorf("Esc-after-r changed text: %q", got)
	}
	if v.operatorPending != 0 {
		t.Errorf("operatorPending should clear: got %q", v.operatorPending)
	}
}

func TestChangeWord_DeletesWordAndEntersInsert(t *testing.T) {
	v, ta := newTestModal(t)
	seedNormal(t, v, ta, "foo bar baz")
	pressNormal(v, ta, "cw")
	if got := ta.Value(); got != "bar baz" {
		t.Errorf("cw on 'foo bar baz' (col 0): got %q, want 'bar baz'", got)
	}
	if v.Mode() != Insert {
		t.Errorf("cw should enter Insert, got %v", v.Mode())
	}
}

func TestDeleteWord_StaysInNormal(t *testing.T) {
	v, ta := newTestModal(t)
	seedNormal(t, v, ta, "foo bar baz")
	pressNormal(v, ta, "dw")
	if got := ta.Value(); got != "bar baz" {
		t.Errorf("dw text: %q, want 'bar baz'", got)
	}
	if v.Mode() != Normal {
		t.Errorf("dw should stay Normal, got %v", v.Mode())
	}
}

func TestChangeToEnd_DollarTruncates(t *testing.T) {
	v, ta := newTestModal(t)
	seedNormal(t, v, ta, "hello world")
	pressNormal(v, ta, "llllll") // col 6
	pressNormal(v, ta, "c$")
	if got := ta.Value(); got != "hello " {
		t.Errorf("c$ at col 6: %q, want 'hello '", got)
	}
}

func TestChangeLine_CC(t *testing.T) {
	v, ta := newTestModal(t)
	seedNormal(t, v, ta, "hello world")
	pressNormal(v, ta, "lllll") // col 5
	pressNormal(v, ta, "cc")
	if got := ta.Value(); got != "" {
		t.Errorf("cc: %q, want empty line", got)
	}
	if v.Mode() != Insert {
		t.Errorf("cc should enter Insert, got %v", v.Mode())
	}
}

func TestFindWordEnd_WordPunctSpace(t *testing.T) {
	cases := []struct {
		line string
		col  int
		want int
	}{
		{"foo bar", 0, 4},
		{"foo bar", 4, 7},
		{"foo,bar", 0, 3},
		{",,foo", 0, 2},
		{"foo", 0, 3},
		{"foo", 5, 5},
	}
	for _, c := range cases {
		got := findWordEnd([]rune(c.line), c.col)
		if got != c.want {
			t.Errorf("findWordEnd(%q, %d): got %d, want %d", c.line, c.col, got, c.want)
		}
	}
}

func TestComposerCursorReadable(t *testing.T) {
	v, ta := newTestModal(t)
	v.SetEnabled(true)
	// Type "hello" in Insert (which passes through to textarea).
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	for _, r := range "hello" {
		consumed, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		if !consumed {
			*ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
	}
	row, col := composerCursor(ta)
	if row != 0 || col != 5 {
		t.Fatalf("cursor after typing 'hello': got (%d,%d), want (0,5) — reflect read broken?", row, col)
	}
}

// regression: typing `cw` shouldn't be parsed as `c` followed by
// `w-motion`. The chord branch has to run BEFORE the motion dispatch.
func TestChordTakesPrecedenceOverMotion(t *testing.T) {
	v, ta := newTestModal(t)
	seedNormal(t, v, ta, "foo bar")
	// If chord didn't take precedence, `w` would move cursor and the
	// `c` would remain pending until timeout, with text untouched.
	pressNormal(v, ta, "cw")
	if got := ta.Value(); got != "bar" {
		t.Errorf("chord-precedence regression: cw produced %q, want 'bar'", got)
	}
	if !strings.HasPrefix(v.Mode().String(), "insert") {
		t.Errorf("after cw: mode=%v, want insert", v.Mode())
	}
}
