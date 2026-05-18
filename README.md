# vimbubble

Modal-style modal editing for [charmbracelet/bubbles](https://github.com/charmbracelet/bubbles) textareas.

Drop it next to an existing `textarea.Model`, route key events through it, get NORMAL/INSERT modes and the verbs you'd expect — `i`, `a`, `o`, `x`, `~`, `r`, `dw`, `cw`, `c$`, `cc`, and friends.

```
go get github.com/guygrigsby/vimbubble
```

## Quick start

```go
import (
    "github.com/charmbracelet/bubbles/textarea"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/guygrigsby/vimbubble"
)

type model struct {
    ta  textarea.Model
    vim *vimbubble.Modal
}

func initial() model {
    ta := textarea.New()
    m := model{ta: ta}
    m.vim = vimbubble.New(&m.ta)
    m.vim.SetEnabled(true)   // off by default; opt in
    return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if k, ok := msg.(tea.KeyMsg); ok {
        // Modal eats the key when it has a binding; otherwise the
        // textarea handles it as usual.
        if consumed, cmds := m.vim.Update(k); consumed {
            return m, tea.Batch(cmds...)
        }
    }
    var cmd tea.Cmd
    m.ta, cmd = m.ta.Update(msg)
    return m, cmd
}

func (m model) View() string {
    return m.vim.ModeLabel() + "\n" + m.ta.View()
}
```

`vim.SetEnabled(false)` disables vim mode entirely — every key passes through to the textarea unmodified, perfect for a `/vim` toggle in your app.

## Supported commands

### NORMAL mode

| Key       | What it does                                 |
|-----------|----------------------------------------------|
| `h` `l`   | cursor left / right                          |
| `j` `k`   | cursor down / up                             |
| `w` `b`   | word forward / backward                      |
| `0` `$`   | line start / end                             |
| `gg` `G`  | buffer top / bottom                          |
| `x`       | delete character forward                     |
| `D`       | delete to end of line                        |
| `~`       | toggle case of char + advance                |
| `r<x>`    | replace char under cursor with `<x>`         |
| `dd`      | delete whole line                            |
| `dw`      | delete word forward                          |
| `d$` `d0` | delete to end / start of line                |
| `cw`      | change word (delete + INSERT)                |
| `c$` `c0` | change to end / start of line                |
| `cc`      | empty the line + INSERT                      |
| `ciw`     | change inside word (any cursor position) + INSERT |
| `caw`     | change a word + its surrounding space + INSERT |
| `diw`     | delete inside word                           |
| `daw`     | delete a word + its surrounding space        |
| `i` `a`   | INSERT at / after cursor                     |
| `I` `A`   | INSERT at line start / end                   |
| `o` `O`   | open new line below / above + INSERT         |
| `/` `:`   | INSERT mode + insert literal `/` (lets host apps reach a slash-command palette without first pressing `i`) |

### INSERT mode

Every key passes through to the textarea, except **Esc** which returns to NORMAL.

## What's not (yet) here

- Visual mode (`v`, `V`)
- Registers (`"a`, `"+`)
- Undo / redo
- Search (`/<query>` as a search, not a passthrough)
- Find-character motions (`f`, `t`, `F`, `T`)
- Text objects (`iw`, `it`, `i"`, …)
- Jump list (`Ctrl+O`, `Ctrl+I`)

These are reasonable next chunks. Open an issue if you want one, or send a PR.

## How it works

vimbubble works by translating NORMAL-mode commands into the key events the textarea already understands (so `h` becomes `KeyLeft`, `w` becomes `Alt+f`, etc.), plus a handful of direct manipulations for verbs the textarea doesn't expose (`~`, `r`, `cw`). The cursor row/col are read via `reflect`+`unsafe` because Bubbles doesn't export them — a regression test (`TestComposerCursorReadable`) pins this contract so a future Bubbles release that renames the fields fails loudly instead of silently no-op'ing.

## License

MIT
