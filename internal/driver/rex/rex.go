package rex

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackkeller/ghostmux/internal/config"
	"github.com/jackkeller/ghostmux/internal/layout"
)

// Shared by Rex Beta and Rex, so it locates the app whichever one is installed.
const bundleID = "com.superlogical.rex"

type Driver struct {
	bin     string
	label   string
	session string
	debug   bool
}

// New returns a driver that builds every window into one new Rex session named label.
func New(label string, debug bool) *Driver {
	return &Driver{label: label, debug: debug}
}

// Available reports whether the rex CLI can be found.
func Available() bool {
	_, err := findCLI()
	return err == nil
}

func findCLI() (string, error) {
	if p, err := exec.LookPath("rex"); err == nil {
		return p, nil
	}
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, "Library", "Application Support", bundleID,
		"RexServer", "Versions", "Current", "Rex Server.app", "Contents", "MacOS", "rex")
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	return "", errors.New("rex CLI not found: install Rex, or add the rex command to your PATH")
}

func (d *Driver) Launch() error {
	bin, err := findCLI()
	if err != nil {
		return err
	}
	d.bin = bin

	client, err := d.appClient()
	if err != nil {
		return err
	}
	if client == "" {
		if err := exec.Command("open", "-b", bundleID).Run(); err != nil {
			return fmt.Errorf("launching Rex: %w", err)
		}
	}
	return nil
}

func (d *Driver) CreateWindow(window config.WindowConfig) error {
	if d.debug {
		fmt.Printf("\n=== Creating window: %s ===\n", window.Name)
	}

	n := len(window.Panes)
	blocks := make([]string, n)

	first, err := d.newWindow(window.Name, d.paneRoot(window, 0))
	if err != nil {
		return err
	}
	blocks[0] = first

	for _, s := range plan(window.Layout, n) {
		if d.debug {
			fmt.Printf("Creating pane %d/%d (%s of pane %d, %d%%)\n", s.pane+1, n, s.dir, s.anchor+1, s.ratio)
		}
		args := []string{"split", "-s", d.session, "-b", blocks[s.anchor],
			"--split=" + s.dir, "--ratio", strconv.Itoa(s.ratio), "--focus=false", "--json"}
		if root := d.paneRoot(window, s.pane); root != "" {
			args = append(args, "-c", root)
		}
		var res struct {
			BlockIDs []string `json:"block_ids"`
		}
		if err := d.runJSON(&res, args...); err != nil {
			return fmt.Errorf("splitting pane %d: %w", s.anchor+1, err)
		}
		if len(res.BlockIDs) == 0 {
			return fmt.Errorf("splitting pane %d: rex returned no block", s.anchor+1)
		}
		blocks[s.pane] = res.BlockIDs[0]
	}

	for i, pane := range window.Panes {
		for _, cmd := range pane.Commands() {
			if d.debug {
				fmt.Printf("Pane %d: %s\n", i, cmd)
			}
			if _, err := d.run("send", "-s", d.session, "-b", blocks[i], cmd); err != nil {
				return err
			}
			if _, err := d.run("send-key", "-s", d.session, "-b", blocks[i], "enter"); err != nil {
				return err
			}
		}
	}

	focus := blocks[0]
	for i, pane := range window.Panes {
		if pane.ShouldFocus() {
			focus = blocks[i]
			break
		}
	}
	if _, err := d.run("focus", "-s", d.session, focus); err != nil {
		return err
	}

	if d.debug {
		fmt.Printf("=== Window '%s' complete ===\n\n", window.Name)
	}

	return nil
}

// newWindow creates the session on first use, then a tab per later window, and
// returns the ID of the tab's initial pane.
func (d *Driver) newWindow(name, root string) (string, error) {
	if d.session == "" {
		args := []string{"new", d.label, "--window", name, "--json"}
		if root != "" {
			args = append(args, "-c", root)
		}
		var res struct {
			SessionID      string `json:"session_id"`
			InitialWindows []struct {
				BlockIDs []string `json:"block_ids"`
			} `json:"initial_windows"`
		}
		if err := d.runJSON(&res, args...); err != nil {
			return "", fmt.Errorf("creating session: %w", err)
		}
		if res.SessionID == "" || len(res.InitialWindows) == 0 || len(res.InitialWindows[0].BlockIDs) == 0 {
			return "", errors.New("creating session: rex returned no session or block")
		}
		d.session = res.SessionID
		d.show()
		return res.InitialWindows[0].BlockIDs[0], nil
	}

	args := []string{"window", "new", "-s", d.session, name, "--json"}
	if root != "" {
		args = append(args, "-c", root)
	}
	var res struct {
		BlockIDs []string `json:"block_ids"`
	}
	if err := d.runJSON(&res, args...); err != nil {
		return "", fmt.Errorf("creating tab: %w", err)
	}
	if len(res.BlockIDs) == 0 {
		return "", errors.New("creating tab: rex returned no block")
	}
	return res.BlockIDs[0], nil
}

// show switches the Rex app to the new session. The layout is still usable from
// the app's session list if this fails, so failures are only reported.
func (d *Driver) show() {
	var client string
	for deadline := time.Now().Add(10 * time.Second); ; {
		c, err := d.appClient()
		if err == nil && c != "" {
			client = c
			break
		}
		if time.Now().After(deadline) {
			fmt.Fprintln(os.Stderr, "warning: no Rex app connected; open Rex to see the session")
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	if _, err := d.run("-C", client, "do", "session.select", "session_id="+d.session); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not show the session in Rex: %v\n", err)
	}
}

// appClient returns the ID of a connected Rex app, preferring the one showing
// the session ghostmux was run from. It returns "" when no app is connected.
func (d *Driver) appClient() (string, error) {
	if s := os.Getenv("REX_SESSION"); s != "" {
		if id, err := d.firstApp("client", "ls", "-s", s, "--json"); err == nil && id != "" {
			return id, nil
		}
	}
	return d.firstApp("client", "ls", "--json")
}

func (d *Driver) firstApp(args ...string) (string, error) {
	var res struct {
		Clients []struct {
			ClientID string `json:"client_id"`
			Info     struct {
				Kind string `json:"kind"`
			} `json:"info"`
		} `json:"clients"`
	}
	if err := d.runJSON(&res, args...); err != nil {
		return "", err
	}
	for _, c := range res.Clients {
		if c.Info.Kind == "app" {
			return c.ClientID, nil
		}
	}
	return "", nil
}

func (d *Driver) paneRoot(window config.WindowConfig, i int) string {
	root := window.Panes[i].GetRoot()
	if root == "" {
		root = window.Root
	}
	if root == "" {
		root, _ = os.Getwd()
		return root
	}
	root = os.ExpandEnv(root)
	if root == "~" || strings.HasPrefix(root, "~/") {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, root[1:])
	}
	return root
}

func (d *Driver) run(args ...string) ([]byte, error) {
	if d.debug {
		fmt.Printf("→ rex %s\n", strings.Join(args, " "))
	}

	out, err := exec.Command(d.bin, args...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("rex %s failed: %w\nOutput: %s", args[0], err, exitErr.Stderr)
		}
		return nil, fmt.Errorf("rex %s failed: %w", args[0], err)
	}

	return out, nil
}

func (d *Driver) runJSON(v any, args ...string) error {
	out, err := d.run(args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(out, v); err != nil {
		return fmt.Errorf("parsing rex %s output: %w", args[0], err)
	}
	return nil
}

// split creates config pane `pane` by splitting the already-created pane
// `anchor`, giving the new pane `ratio` percent of the anchor's space.
type split struct {
	pane, anchor int
	dir          string
	ratio        int
}

// plan returns the splits that build layoutType for n panes. Rex splits any pane
// by ID with an explicit ratio, so unlike Ghostty no focus navigation or
// equalizing pass is needed.
func plan(layoutType string, n int) []split {
	var steps []split

	if layoutType == "tiled" {
		rowsFor := layout.TiledGrid(n)
		colStart := make([]int, len(rowsFor))
		for c := 1; c < len(rowsFor); c++ {
			colStart[c] = colStart[c-1] + rowsFor[c-1]
		}
		for c := 1; c < len(rowsFor); c++ {
			steps = append(steps, split{colStart[c], colStart[c-1], "right", evenRatio(len(rowsFor) - c + 1)})
		}
		for c, rows := range rowsFor {
			for r := 1; r < rows; r++ {
				steps = append(steps, split{colStart[c] + r, colStart[c] + r - 1, "below", evenRatio(rows - r + 1)})
			}
		}
		return steps
	}

	strategy := layout.NewStrategy(layoutType, n)
	for i := 1; i < n; i++ {
		dir := "below"
		if strategy.NextSplit(i) == "horizontal" {
			dir = "right"
		}
		ratio := 50
		if layoutType == "even-horizontal" || layoutType == "even-vertical" {
			ratio = evenRatio(n - i + 1)
		}
		steps = append(steps, split{i, i - 1, dir, ratio})
	}
	return steps
}

// evenRatio is the share a new pane takes when splitting a pane that must end
// up holding `remaining` equal panes, so that the anchor keeps exactly one share.
func evenRatio(remaining int) int {
	return 100 * (remaining - 1) / remaining
}
