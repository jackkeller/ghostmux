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

func (d *AppleScriptDriver) sendKeyCode(keyCode int, modifiers []string) error {
	modStr := ""
	if len(modifiers) > 0 {
		modStr = fmt.Sprintf(" using {%s}", strings.Join(modifiers, ", "))
	}

	script := fmt.Sprintf(`
		tell application "System Events"
			tell process "Ghostty"
				key code %d%s
			end tell
		end tell
	`, keyCode, modStr)

	return d.runScript(script)
}

func (d *AppleScriptDriver) splitHorizontal() error {
	return d.sendKeystroke("d", []string{"command down"})
}

func (d *AppleScriptDriver) splitVertical() error {
	return d.sendKeystroke("d", []string{"command down", "shift down"})
}

// goto_split:previous — default keybinding is Cmd+[ (key code 33)
func (d *AppleScriptDriver) gotoPreviousSplit() error {
	return d.sendKeyCode(33, []string{"command down"})
}

// goto_split:next — default keybinding is Cmd+] (key code 30)
func (d *AppleScriptDriver) gotoNextSplit() error {
	return d.sendKeyCode(30, []string{"command down"})
}
