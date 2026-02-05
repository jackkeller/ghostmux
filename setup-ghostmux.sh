#!/bin/bash
# setup-ghostmux.sh - Run this to create the entire project structure

DEFAULT_DIR="$HOME/code/ghostmux"

printf "Where should ghostmux be installed? [%s]: " "$DEFAULT_DIR"
read -r PROJECT_DIR
PROJECT_DIR="${PROJECT_DIR:-$DEFAULT_DIR}"

# Expand ~ to $HOME
PROJECT_DIR="${PROJECT_DIR/#\~/$HOME}"

echo "Creating ghostmux project at $PROJECT_DIR..."

# Create directory structure
mkdir -p "$PROJECT_DIR"/{cmd/ghostmux,internal/{config,driver/ghostty,layout},examples}

cd "$PROJECT_DIR"

# Create go.mod
cat > go.mod <<'EOF'
module github.com/jackkeller/ghostmux

go 1.21

require gopkg.in/yaml.v3 v3.0.1
EOF

# Create cmd/ghostmux/main.go
cat > cmd/ghostmux/main.go <<'EOF'
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackkeller/ghostmux/internal/config"
	"github.com/jackkeller/ghostmux/internal/driver/ghostty"
)

const version = "0.1.0"

func main() {
	var (
		configPath  = flag.String("config", "", "Path to config file")
		configName  = flag.String("name", "", "Name of config in ~/.ghostmux/")
		listConfigs = flag.Bool("list", false, "List available configs")
		showVersion = flag.Bool("version", false, "Show version")
		debug       = flag.Bool("debug", false, "Enable debug logging")
		dryRun      = flag.Bool("dry-run", false, "Validate config without launching")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("ghostmux v%s\n", version)
		os.Exit(0)
	}

	configsDir := filepath.Join(os.Getenv("HOME"), ".ghostmux")

	if *listConfigs {
		listAvailableConfigs(configsDir)
		return
	}

	// Determine config path
	cfgPath := determineConfigPath(*configPath, *configName, configsDir)

	// Load config
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if *debug {
		fmt.Printf("Loaded config: %s (%d window(s))\n", cfgPath, len(cfg.Windows))
	}

	if *dryRun {
		fmt.Printf("✓ Config valid: %s\n", cfgPath)
		for i, w := range cfg.Windows {
			fmt.Printf("  Window %d: %s (%d panes)\n", i+1, w.Name, len(w.Panes))
		}
		os.Exit(0)
	}

	// Create driver
	drv := ghostty.New(*debug)

	// Launch Ghostty
	if *debug {
		fmt.Println("Launching Ghostty...")
	}
	if err := drv.Launch(); err != nil {
		fmt.Fprintf(os.Stderr, "Error launching Ghostty: %v\n", err)
		os.Exit(1)
	}

	// Create windows
	for i, window := range cfg.Windows {
		if *debug {
			fmt.Printf("Creating window %d/%d: %s\n", i+1, len(cfg.Windows), window.Name)
		}
		if err := drv.CreateWindow(window); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating window '%s': %v\n", window.Name, err)
			os.Exit(1)
		}
	}

	if *debug {
		fmt.Println("✨ Done!")
	}
}

func determineConfigPath(configPath, configName, configsDir string) string {
	if configName != "" {
		return filepath.Join(configsDir, configName+".yml")
	}
	if configPath != "" {
		return configPath
	}
	return ".ghostmux.yml"
}

func listAvailableConfigs(dir string) {
	files, err := filepath.Glob(filepath.Join(dir, "*.yml"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing configs: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Printf("No configs found in %s\n", dir)
		fmt.Println("\nCreate one with:")
		fmt.Printf("  mkdir -p %s\n", dir)
		fmt.Printf("  vim %s/my-project.yml\n", dir)
		return
	}

	fmt.Println("Available configs:")
	for _, file := range files {
		name := filepath.Base(file)
		name = name[:len(name)-4]
		fmt.Printf("  • %s\n", name)
	}
	fmt.Printf("\nLaunch with: ghostmux --name <config-name>\n")
}
EOF

# Create internal/config/config.go
cat > internal/config/config.go <<'EOF'
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Windows []WindowConfig `yaml:"windows"`
}

type WindowConfig struct {
	Name   string      `yaml:"name"`
	Root   string      `yaml:"root,omitempty"`
	Layout string      `yaml:"layout,omitempty"`
	Panes  []PaneEntry `yaml:"panes"`
}

type PaneEntry struct {
	raw interface{}
}

type PaneConfig struct {
	Commands []string `yaml:"commands,omitempty"`
	Focus    bool     `yaml:"focus,omitempty"`
	Root     string   `yaml:"root,omitempty"`
}

func (p *PaneEntry) UnmarshalYAML(value *yaml.Node) error {
	var str string
	if err := value.Decode(&str); err == nil {
		p.raw = str
		return nil
	}

	var cfg PaneConfig
	if err := value.Decode(&cfg); err != nil {
		return fmt.Errorf("pane must be a string or object with 'commands': %w", err)
	}
	p.raw = cfg
	return nil
}

func (p *PaneEntry) Commands() []string {
	switch v := p.raw.(type) {
	case string:
		return []string{v}
	case PaneConfig:
		return v.Commands
	default:
		return []string{}
	}
}

func (p *PaneEntry) GetRoot() string {
	if cfg, ok := p.raw.(PaneConfig); ok {
		return cfg.Root
	}
	return ""
}

func (p *PaneEntry) ShouldFocus() bool {
	if cfg, ok := p.raw.(PaneConfig); ok {
		return cfg.Focus
	}
	return false
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if len(c.Windows) == 0 {
		return fmt.Errorf("config must have at least one window")
	}

	for i, window := range c.Windows {
		if window.Name == "" {
			return fmt.Errorf("window %d: missing name", i)
		}
		if len(window.Panes) == 0 {
			return fmt.Errorf("window '%s': must have at least one pane", window.Name)
		}

		if window.Layout != "" {
			validLayouts := map[string]bool{
				"tiled":           true,
				"even-horizontal": true,
				"even-vertical":   true,
				"alternating":     true,
			}
			if !validLayouts[window.Layout] {
				return fmt.Errorf("window '%s': invalid layout '%s'", window.Name, window.Layout)
			}
		}
	}

	return nil
}
EOF

# Create internal/driver/ghostty/applescript.go
cat > internal/driver/ghostty/applescript.go <<'EOF'
package ghostty

import (
	"fmt"
	"os/exec"
	"strings"
)

type AppleScriptDriver struct {
	debug bool
}

func NewAppleScript(debug bool) *AppleScriptDriver {
	return &AppleScriptDriver{debug: debug}
}

func (d *AppleScriptDriver) runScript(script string) error {
	if d.debug {
		fmt.Printf("→ AppleScript:\n%s\n", strings.TrimSpace(script))
	}

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("AppleScript failed: %w\nOutput: %s", err, output)
	}

	if d.debug && len(output) > 0 {
		fmt.Printf("← Output: %s\n", output)
	}

	return nil
}

func (d *AppleScriptDriver) activateGhostty() error {
	script := `tell application "Ghostty" to activate`
	return d.runScript(script)
}

func (d *AppleScriptDriver) sendKeystroke(key string, modifiers []string) error {
	modStr := ""
	if len(modifiers) > 0 {
		modStr = fmt.Sprintf(" using {%s}", strings.Join(modifiers, ", "))
	}

	script := fmt.Sprintf(`
		tell application "System Events"
			tell process "Ghostty"
				keystroke "%s"%s
			end tell
		end tell
	`, key, modStr)

	return d.runScript(script)
}

func (d *AppleScriptDriver) typeText(text string) error {
	escaped := strings.ReplaceAll(text, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)

	script := fmt.Sprintf(`
		tell application "System Events"
			tell process "Ghostty"
				keystroke "%s"
				key code 36
			end tell
		end tell
	`, escaped)

	return d.runScript(script)
}

func (d *AppleScriptDriver) splitHorizontal() error {
	return d.sendKeystroke("d", []string{"command down"})
}

func (d *AppleScriptDriver) splitVertical() error {
	return d.sendKeystroke("d", []string{"command down", "shift down"})
}

func (d *AppleScriptDriver) navigateLeft() error {
	return d.sendKeystroke("h", []string{"command down", "shift down"})
}

func (d *AppleScriptDriver) navigateRight() error {
	return d.sendKeystroke("l", []string{"command down", "shift down"})
}

func (d *AppleScriptDriver) navigateUp() error {
	return d.sendKeystroke("k", []string{"command down", "shift down"})
}

func (d *AppleScriptDriver) navigateDown() error {
	return d.sendKeystroke("j", []string{"command down", "shift down"})
}
EOF

# Create internal/driver/ghostty/ghostty.go
cat > internal/driver/ghostty/ghostty.go <<'EOF'
package ghostty

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/jackkeller/ghostmux/internal/config"
	"github.com/jackkeller/ghostmux/internal/layout"
)

type Driver struct {
	as    *AppleScriptDriver
	debug bool
}

type panePosition struct {
	row int
	col int
}

func New(debug bool) *Driver {
	return &Driver{
		as:    NewAppleScript(debug),
		debug: debug,
	}
}

func (d *Driver) Launch() error {
	cmd := exec.Command("open", "-a", "Ghostty")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("launching Ghostty: %w", err)
	}

	time.Sleep(1500 * time.Millisecond)

	return d.as.activateGhostty()
}

func (d *Driver) CreateWindow(window config.WindowConfig) error {
	if d.debug {
		fmt.Printf("\n=== Creating window: %s ===\n", window.Name)
	}

	if window.Root != "" {
		if d.debug {
			fmt.Printf("Changing to: %s\n", window.Root)
		}
		if err := d.as.typeText(fmt.Sprintf("cd %s", window.Root)); err != nil {
			return err
		}
		time.Sleep(200 * time.Millisecond)

		if err := d.as.typeText("clear"); err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}

	layoutStrategy := layout.NewStrategy(window.Layout, len(window.Panes))
	positions := []panePosition{{0, 0}}

	for i := 1; i < len(window.Panes); i++ {
		direction := layoutStrategy.NextSplit(i)

		if d.debug {
			fmt.Printf("Creating pane %d/%d (%s split)\n", i+1, len(window.Panes), direction)
		}

		if direction == "horizontal" {
			if err := d.as.splitHorizontal(); err != nil {
				return fmt.Errorf("splitting horizontally: %w", err)
			}
			lastPos := positions[len(positions)-1]
			positions = append(positions, panePosition{lastPos.row + 1, lastPos.col})
		} else {
			if err := d.as.splitVertical(); err != nil {
				return fmt.Errorf("splitting vertically: %w", err)
			}
			lastPos := positions[len(positions)-1]
			positions = append(positions, panePosition{lastPos.row, lastPos.col + 1})
		}

		time.Sleep(400 * time.Millisecond)
	}

	currentPos := positions[len(positions)-1]

	for i := len(window.Panes) - 1; i >= 0; i-- {
		targetPos := positions[i]

		if err := d.navigateToPane(currentPos, targetPos); err != nil {
			return err
		}
		currentPos = targetPos

		pane := window.Panes[i]

		if paneRoot := pane.GetRoot(); paneRoot != "" {
			if d.debug {
				fmt.Printf("Pane %d: cd %s\n", i, paneRoot)
			}
			if err := d.as.typeText(fmt.Sprintf("cd %s", paneRoot)); err != nil {
				return err
			}
			time.Sleep(100 * time.Millisecond)
		}

		commands := pane.Commands()
		for _, cmd := range commands {
			if d.debug {
				fmt.Printf("Pane %d: %s\n", i, cmd)
			}
			if err := d.as.typeText(cmd); err != nil {
				return err
			}
			time.Sleep(200 * time.Millisecond)
		}
	}

	if d.debug {
		fmt.Printf("=== Window '%s' complete ===\n\n", window.Name)
	}

	return nil
}

func (d *Driver) navigateToPane(from, to panePosition) error {
	rowDiff := to.row - from.row
	for i := 0; i < abs(rowDiff); i++ {
		if rowDiff > 0 {
			if err := d.as.navigateDown(); err != nil {
				return err
			}
		} else {
			if err := d.as.navigateUp(); err != nil {
				return err
			}
		}
		time.Sleep(150 * time.Millisecond)
	}

	colDiff := to.col - from.col
	for i := 0; i < abs(colDiff); i++ {
		if colDiff > 0 {
			if err := d.as.navigateRight(); err != nil {
				return err
			}
		} else {
			if err := d.as.navigateLeft(); err != nil {
				return err
			}
		}
		time.Sleep(150 * time.Millisecond)
	}

	return nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
EOF

# Create internal/layout/layout.go
cat > internal/layout/layout.go <<'EOF'
package layout

type Strategy interface {
	NextSplit(index int) string
}

type tiledStrategy struct {
	totalPanes int
}

type evenHorizontalStrategy struct{}
type evenVerticalStrategy struct{}
type alternatingStrategy struct{}

func NewStrategy(layoutType string, totalPanes int) Strategy {
	switch layoutType {
	case "tiled":
		return &tiledStrategy{totalPanes: totalPanes}
	case "even-horizontal":
		return &evenHorizontalStrategy{}
	case "even-vertical":
		return &evenVerticalStrategy{}
	default:
		return &alternatingStrategy{}
	}
}

func (s *alternatingStrategy) NextSplit(index int) string {
	if index%2 == 0 {
		return "vertical"
	}
	return "horizontal"
}

func (s *evenHorizontalStrategy) NextSplit(index int) string {
	return "horizontal"
}

func (s *evenVerticalStrategy) NextSplit(index int) string {
	return "vertical"
}

func (s *tiledStrategy) NextSplit(index int) string {
	if index <= s.totalPanes/2 {
		return "vertical"
	}
	return "horizontal"
}
EOF

# Create examples/simple.yml
cat > examples/simple.yml <<'EOF'
windows:
  - name: simple-dev
    root: ~/code
    layout: alternating
    panes:
      - git status
      - ls -la
      - echo "Ready to code!"
EOF

# Create examples/panda.yml
cat > examples/panda.yml <<'EOF'
windows:
  - name: panda-cms
    root: ~/code/panda-cms
    layout: tiled
    panes:
      - commands:
          - npm run dev
      - commands:
          - git status
          - git log --oneline -5
      - nvim
      - commands:
          - npm run test:watch
        focus: true
EOF

# Create Makefile
cat > Makefile <<'EOF'
.PHONY: build install test clean run dev validate release setup-examples

build:
	@echo "Building ghostmux..."
	@go build -o bin/ghostmux cmd/ghostmux/main.go
	@echo "✓ Binary created: bin/ghostmux"

install:
	@echo "Installing ghostmux..."
	@go install ./cmd/ghostmux
	@echo "✓ Installed to $(shell go env GOPATH)/bin/ghostmux"

test:
	@go test ./...

clean:
	@rm -rf bin/
	@echo "✓ Cleaned bin/"

run:
	@go run cmd/ghostmux/main.go --debug --config examples/simple.yml

dev: build
	@./bin/ghostmux --debug --config examples/simple.yml

validate:
	@echo "Validating example configs..."
	@go run cmd/ghostmux/main.go --dry-run --config examples/simple.yml
	@go run cmd/ghostmux/main.go --dry-run --config examples/panda.yml
	@echo "✓ All configs valid"

release:
	@echo "Building release binaries..."
	@mkdir -p bin/release
	@GOOS=darwin GOARCH=arm64 go build -o bin/release/ghostmux-darwin-arm64 cmd/ghostmux/main.go
	@GOOS=darwin GOARCH=amd64 go build -o bin/release/ghostmux-darwin-amd64 cmd/ghostmux/main.go
	@echo "✓ Release binaries in bin/release/"

setup-examples:
	@mkdir -p ~/.ghostmux
	@cp examples/*.yml ~/.ghostmux/
	@echo "✓ Example configs installed to ~/.ghostmux/"
EOF

# Create README.md
cat > README.md <<'EOF'
# ghostmux

iTerm2's itermocil for Ghostty - Launch complex terminal layouts with one command.

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
ghostmux --name dev
```

## Usage
```bash
ghostmux --name my-project      # Launch from config name
ghostmux --config /path/to.yml  # Launch from config path
ghostmux                        # Launch from .ghostmux.yml in current dir
ghostmux --list                 # List available configs
ghostmux --dry-run --name test  # Validate config
ghostmux --debug --name test    # Debug mode
```

## Configuration

See examples/ directory for sample configs.

## Requirements

- macOS (uses AppleScript)
- Ghostty terminal

## License

MIT
EOF

# Create .gitignore
cat > .gitignore <<'EOF'
bin/
*.swp
*.swo
.DS_Store
EOF

# Initialize and tidy
go mod tidy

echo ""
echo "✅ Project created successfully!"
echo ""
echo "Next steps:"
echo "  cd $PROJECT_DIR"
echo "  make validate    # Check configs are valid"
echo "  make build       # Build the binary"
echo "  make dev         # Test it out!"
echo ""
EOF

# Make it executable
chmod +x setup-ghostmux.sh

# Save the script
cat setup-ghostmux.sh
