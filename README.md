# ghostmux

iTerm2's [itermocil](https://github.com/TomAnthony/itermocil) for [Ghostty](https://ghostty.org) and Rex - Launch complex terminal layouts with one command.

> **macOS only** — ghostmux uses AppleScript to control Ghostty and the `rex` CLI to control Rex. It is not compatible with Linux or Windows.

## Installation
```bash
go install github.com/jackkeller/ghostmux/cmd/ghostmux@latest
```

Or build from source:
```bash
git clone https://github.com/jackkeller/ghostmux
cd ghostmux
make install
```

## Quick Start
```bash
# Create config directory
mkdir -p ~/.ghostmux

# Create your first layout
cat > ~/.ghostmux/dev.yml <<'YAML'
windows:
  - name: my-project
    root: ~/code/my-project
    layout: tiled
    panes:
      - npm run dev
      - git status
      - nvim
YAML

# Launch it
ghostmux dev
```

## Usage
```bash
ghostmux dev                    # Launch from config name
ghostmux --config /path/to.yml  # Launch from config path
ghostmux                        # Launch from .ghostmux.yml in current dir
ghostmux --list                 # List available configs
ghostmux --dry-run dev          # Validate config
ghostmux --debug dev            # Debug mode
ghostmux --terminal rex dev     # Force a terminal: auto (default), ghostty, or rex
```

### Ghostty vs. Rex

With `--terminal auto`, ghostmux drives Rex when run from inside Rex (`REX_SESSION` is set) and Ghostty otherwise.

In Rex, each launch creates a new session named after the config (`ghostmux dev` → session `dev`). Each window in the config becomes a tab and each pane becomes a split. The Rex app then switches to that session. Rex builds splits through its CLI, so layouts come out exactly sized and nothing is typed into the focused window. Set `focus: true` on a pane to focus it; the first pane is focused otherwise. Rex Beta and Rex are both supported.

## Layouts

Set `layout` on a window to control how panes are arranged. Examples shown with 3 panes:

**`tiled`** - Top row, then splits bottom row into columns
```
┌──────────────────┐
│        1         │
├────────┬─────────┤
│   2    │    3    │
└────────┴─────────┘
```

**`even-horizontal`** - All panes side by side as columns
```
┌──────┬──────┬──────┐
│  1   │  2   │  3   │
└──────┴──────┴──────┘
```

**`even-vertical`** - All panes stacked as rows
```
┌────────────────────┐
│         1          │
├────────────────────┤
│         2          │
├────────────────────┤
│         3          │
└────────────────────┘
```

**`grid`** - 2x2 grid using `tiled` with 4 panes
```
┌────────┬─────────┐
│   1    │    2    │
├────────┼─────────┤
│   3    │    4    │
└────────┴─────────┘
```

**`two-column`** - Side-by-side split using `even-horizontal` with 2 panes
```
┌─────────┬─────────┐
│         │         │
│    1    │    2    │
│         │         │
└─────────┴─────────┘
```

**default (alternating)** - Alternates between right and down splits
```
┌─────────┬─────────┐
│         │    2    │
│    1    ├─────────┤
│         │    3    │
└─────────┴─────────┘
```

See `examples/` directory for full sample configs.

## Configuration

## Requirements

- [Go](https://go.dev) 1.21+
- macOS (uses AppleScript)
- [Ghostty](https://ghostty.org) or Rex (Beta or later)

## License

MIT
