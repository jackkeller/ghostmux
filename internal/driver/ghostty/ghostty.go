package ghostty

import (
	"fmt"
	"math"
	"os/exec"
	"time"

	"github.com/jackkeller/ghostmux/internal/config"
	"github.com/jackkeller/ghostmux/internal/layout"
)

type Driver struct {
	as    *AppleScriptDriver
	debug bool
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

	// "tiled" needs a real grid: Ghostty only ever splits the focused pane, so we
	// must reposition focus between splits rather than just picking a direction.
	if window.Layout == "tiled" {
		if err := d.buildTiledGrid(len(window.Panes)); err != nil {
			return fmt.Errorf("building tiled grid: %w", err)
		}
	} else {
		layoutStrategy := layout.NewStrategy(window.Layout, len(window.Panes))

		for i := 1; i < len(window.Panes); i++ {
			direction := layoutStrategy.NextSplit(i)

			if d.debug {
				fmt.Printf("Creating pane %d/%d (%s split)\n", i+1, len(window.Panes), direction)
			}

			if direction == "horizontal" {
				if err := d.as.splitHorizontal(); err != nil {
					return fmt.Errorf("splitting horizontally: %w", err)
				}
			} else {
				if err := d.as.splitVertical(); err != nil {
					return fmt.Errorf("splitting vertically: %w", err)
				}
			}

			time.Sleep(400 * time.Millisecond)
		}
	}

	// Navigate back to the first pane, then walk forward using goto_split:next.
	// This ensures commands that steal focus (e.g. "code .") run last.
	for j := len(window.Panes) - 1; j > 0; j-- {
		if err := d.as.gotoPreviousSplit(); err != nil {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}

	for i := 0; i < len(window.Panes); i++ {
		if i > 0 {
			if err := d.as.activateGhostty(); err != nil {
				return err
			}
			time.Sleep(300 * time.Millisecond)

			if d.debug {
				fmt.Printf("Navigating to pane %d (goto_split:next)\n", i)
			}
			if err := d.as.gotoNextSplit(); err != nil {
				return err
			}
			time.Sleep(200 * time.Millisecond)
		}

		pane := window.Panes[i]

		paneRoot := pane.GetRoot()
		if paneRoot == "" {
			paneRoot = window.Root
		}
		if paneRoot != "" && i != 0 {
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

// buildTiledGrid arranges n panes into a balanced grid. Ghostty's new_split only
// splits the currently focused pane (and moves focus to the new one), so unlike
// tmux there is no global re-layout — we build columns first, then split each
// column into rows, navigating focus spatially between steps. Stacked right-splits
// start unequal, so we equalize sizes at the end.
func (d *Driver) buildTiledGrid(n int) error {
	if n <= 1 {
		return nil
	}

	cols := int(math.Ceil(math.Sqrt(float64(n))))
	rowsFor := make([]int, cols)
	for c := 0; c < cols; c++ {
		rowsFor[c] = n / cols
		if c < n%cols {
			rowsFor[c]++
		}
	}

	if d.debug {
		fmt.Printf("Tiled grid: %d panes → %d columns %v rows\n", n, cols, rowsFor)
	}

	// 1. Create the columns across the top (new_split:right).
	for c := 1; c < cols; c++ {
		if err := d.as.splitHorizontal(); err != nil {
			return fmt.Errorf("creating column %d: %w", c, err)
		}
		time.Sleep(400 * time.Millisecond)
	}

	// Focus is on the rightmost column; walk back to the leftmost.
	for c := 1; c < cols; c++ {
		if err := d.as.gotoSplitLeft(); err != nil {
			return err
		}
		time.Sleep(150 * time.Millisecond)
	}

	// 2. Split each column into rows (new_split:down), then step right.
	for c := 0; c < cols; c++ {
		for r := 1; r < rowsFor[c]; r++ {
			if err := d.as.splitVertical(); err != nil {
				return fmt.Errorf("creating row %d in column %d: %w", r, c, err)
			}
			time.Sleep(400 * time.Millisecond)
		}
		if c < cols-1 {
			if err := d.as.gotoSplitRight(); err != nil {
				return err
			}
			time.Sleep(150 * time.Millisecond)
		}
	}

	// 3. Even out pane sizes.
	time.Sleep(150 * time.Millisecond)
	return d.as.equalizeSplits()
}
