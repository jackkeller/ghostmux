# ghostmux

iTerm2's [itermocil](https://github.com/TomAnthony/itermocil) for [Ghostty](https://ghostty.org) - Launch complex terminal layouts with one command.

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
```

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

- macOS (uses AppleScript)
- Ghostty terminal

## License

MIT
