# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Go library that adds vim-style modal editing (NORMAL / INSERT) to a `charmbracelet/bubbles` `textarea.Model`. Single package, no `main`. Consumer wires `Modal.Update` ahead of `textarea.Update` and the modal eats keys it has bindings for.

## Commands

```
go test ./...                          # full suite (tests live in vim_test.go)
go test -run TestChangeInsideWord ./... # single test
go vet ./...
go build ./...
```

No linter config, no CI, no example binary. The tests are the contract.

## Architecture

Three files do most of the work, each with a distinct job:

- `vim.go` — public surface (`Modal`, `Mode`, `New`, `Update`, `SetEnabled`, `SetMode`, `ModeLabel`) plus the `chord` state machine for multi-key sequences.
- `normal.go` — NORMAL-mode dispatcher. `handleNormal` is the single entry point; `dispatch2Key` (cw, dd, gg, d$, …) and `dispatchTextObject` (ciw, caw, diw, daw) finish chords.
- `cursor.go` — buffer-level primitives: cursor read, character replace, word-bound math (`wordBoundsInner` / `wordBoundsAround`), range delete.

### The synthesis trick

`Modal` does NOT own a cursor model. Most NORMAL commands translate into the `tea.KeyMsg` events the textarea already understands (`h` → `KeyLeft`, `w` → `Alt+f`, `G` → `Alt+>`) and forward them to `*v.composer`. The textarea remains the source of truth for cursor + word walks. Only verbs the textarea can't express (`~`, `r`, `cw`, `ciw`, etc.) reach into the buffer directly via `cursor.go` helpers.

### The reflect+unsafe contract

`composerCursor` reads the textarea's unexported `row` and `col` fields via `reflect`+`unsafe`. This is the most fragile thing in the codebase. A bubbles release that renames those fields would silently break `~`, `r`, `cw`, `ciw`, `caw`, `cc`, `dd`, and friends. `TestComposerCursorReadable` pins the contract so the breakage is loud. If you bump the bubbles dependency, run the tests; if that test fails, look at the bubbles `textarea.Model` struct before touching anything else.

### Chord state

Two shapes of multi-key sequences flow through `chord`:

- 2-key operator+motion: `cw`, `dd`, `d$`, `gg`, `cc`, `c0`, `d0`.
- 3-key operator+specifier+object: `ciw`, `caw`, `diw`, `daw`. `i` / `a` after a pending `c` / `d` become specifiers (NOT mode switches) — this is the branch in `handleNormal` that makes text objects parse.

Chord state ages out after `chordTimeout` (800ms) so a stray `g` doesn't trap the composer. `operatorPending` is a separate one-shot for `r<x>` (replace).

### Update contract

`Modal.Update(msg) (consumed bool, cmds []tea.Cmd)` — if `consumed`, host must NOT forward the key to its textarea. In `Disabled` mode `consumed` is always false (modal is invisible). In `Insert` mode every key passes through (returns false) except `Esc` which flips to NORMAL.

## What's not implemented

Visual mode, registers, undo/redo, real `/<query>` search, jump list, find-character motions (`f`/`t`/`F`/`T`), bracket/quote text objects (`i"`, `it`, `i(`). Adding bracket-based text objects would slot in beside `wordBoundsInner`/`wordBoundsAround` in `cursor.go` and a new case in `dispatchTextObject`.

`/` and `:` in NORMAL flip to INSERT and pass the literal char through, so host apps can implement their own slash-command palette without forcing users through `i` first.
