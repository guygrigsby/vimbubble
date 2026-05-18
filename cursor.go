package vimbubble

import (
	"reflect"
	"strings"
	"unicode"
	"unsafe"

	"github.com/charmbracelet/bubbles/textarea"
)

// composerCursor returns the textarea's current (row, col). Bubbles
// doesn't expose these — they live as unexported `row` and `col` ints
// on textarea.Model — so we cheat with reflect + unsafe to read them.
// Fragile if bubbles renames or restructures, but adequately pinned
// by TestComposerCursorReadable in the test suite. The alternative
// is maintaining a shadow cursor that intercepts every key event
// the textarea processes; the bookkeeping bloat isn't worth it.
func composerCursor(t *textarea.Model) (row, col int) {
	v := reflect.ValueOf(t).Elem()
	rowF := v.FieldByName("row")
	colF := v.FieldByName("col")
	if !rowF.IsValid() || !colF.IsValid() {
		return 0, 0
	}
	row = *(*int)(unsafe.Pointer(rowF.UnsafeAddr()))
	col = *(*int)(unsafe.Pointer(colF.UnsafeAddr()))
	return
}

// replaceCharAtCursor edits the textarea's value so the character at
// the current cursor position is replaced with `r`. Then re-pins the
// cursor to its original column on the same line. Returns true when
// a replacement happened, false when the cursor was past the end of
// its line (nothing to replace).
func replaceCharAtCursor(t *textarea.Model, replacement rune) bool {
	row, col := composerCursor(t)
	lines := strings.Split(t.Value(), "\n")
	if row < 0 || row >= len(lines) {
		return false
	}
	runes := []rune(lines[row])
	if col < 0 || col >= len(runes) {
		return false
	}
	runes[col] = replacement
	lines[row] = string(runes)
	t.SetValue(strings.Join(lines, "\n"))
	moveCursorTo(t, row, col)
	return true
}

// moveCursorTo positions the textarea cursor at (row, col). Uses the
// public API (CursorUp / CursorDown / SetCursor) so a future Bubbles
// refactor that reshapes the internals doesn't break this.
func moveCursorTo(t *textarea.Model, row, col int) {
	curRow, _ := composerCursor(t)
	for curRow > 0 {
		t.CursorUp()
		curRow--
	}
	for i := 0; i < row; i++ {
		t.CursorDown()
	}
	t.SetCursor(col)
}

// toggleCaseAtCursor implements vim's `~`: flips the case of the
// character at the cursor and advances one cell. No-op when the
// cursor is past the end of its line.
func toggleCaseAtCursor(t *textarea.Model) {
	row, col := composerCursor(t)
	lines := strings.Split(t.Value(), "\n")
	if row < 0 || row >= len(lines) {
		return
	}
	runes := []rune(lines[row])
	if col < 0 || col >= len(runes) {
		return
	}
	r := runes[col]
	switch {
	case unicode.IsLower(r):
		runes[col] = unicode.ToUpper(r)
	case unicode.IsUpper(r):
		runes[col] = unicode.ToLower(r)
	}
	lines[row] = string(runes)
	t.SetValue(strings.Join(lines, "\n"))
	if col+1 >= len(runes) && row+1 < len(lines) {
		moveCursorTo(t, row+1, 0)
		return
	}
	moveCursorTo(t, row, col+1)
}

// charKind classifies a rune for vim's word-motion semantics.
type charKind int

const (
	classSpace charKind = iota
	classWord
	classPunct
)

func charClass(r rune) charKind {
	switch {
	case r == ' ' || r == '\t':
		return classSpace
	case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
		return classWord
	default:
		return classPunct
	}
}

// findWordEnd returns the column AFTER the current word starting at
// `col` on `line`. Mirrors vim's `w` motion: word characters are
// alnum + _; a run of punctuation also counts as a word boundary;
// one trailing space gets swallowed so `cw foo` reaches `bar` cleanly
// (vim's quirk).
func findWordEnd(line []rune, col int) int {
	if col >= len(line) {
		return col
	}
	i := col
	startKind := charClass(line[i])
	for i < len(line) && charClass(line[i]) == startKind && startKind != classSpace {
		i++
	}
	if i < len(line) && line[i] == ' ' {
		i++
	}
	return i
}

// deleteRangeOnRow removes runes [startCol, endCol) on `row`. Sets
// the cursor at startCol. No-op when the range is empty or outside
// the line.
func deleteRangeOnRow(t *textarea.Model, row, startCol, endCol int) {
	if endCol <= startCol {
		return
	}
	lines := strings.Split(t.Value(), "\n")
	if row < 0 || row >= len(lines) {
		return
	}
	runes := []rune(lines[row])
	if startCol < 0 {
		startCol = 0
	}
	if endCol > len(runes) {
		endCol = len(runes)
	}
	if startCol >= len(runes) {
		return
	}
	lines[row] = string(runes[:startCol]) + string(runes[endCol:])
	t.SetValue(strings.Join(lines, "\n"))
	moveCursorTo(t, row, startCol)
}

// clearRow empties the content of `row` without removing the row
// itself. Used by `cc` (change line) where vim leaves the user on a
// blank line in INSERT mode.
func clearRow(t *textarea.Model, row int) {
	lines := strings.Split(t.Value(), "\n")
	if row < 0 || row >= len(lines) {
		return
	}
	lines[row] = ""
	t.SetValue(strings.Join(lines, "\n"))
	moveCursorTo(t, row, 0)
}

// deleteWord deletes from the cursor through the end of the current
// word on the current line. Shared by `dw` and `cw` — same range,
// different mode-end behaviour (dw stays Normal, cw enters Insert).
func (v *Modal) deleteWord() {
	row, col := composerCursor(v.composer)
	lines := strings.Split(v.composer.Value(), "\n")
	if row < 0 || row >= len(lines) {
		return
	}
	runes := []rune(lines[row])
	end := findWordEnd(runes, col)
	deleteRangeOnRow(v.composer, row, col, end)
}
