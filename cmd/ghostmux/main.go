package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/jackkeller/ghostmux/internal/config"
	"github.com/jackkeller/ghostmux/internal/driver/ghostty"
	"github.com/jackkeller/ghostmux/internal/driver/rex"
)

type driver interface {
	Launch() error
	CreateWindow(window config.WindowConfig) error
}

var version = "dev"

func init() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
	}
}

func main() {
	var (
		configPath  = flag.String("config", "", "Path to config file")
		configName  = flag.String("name", "", "Name of config in ~/.ghostmux/")
		listConfigs = flag.Bool("list", false, "List available configs")
		showVersion = flag.Bool("version", false, "Show version")
		debug       = flag.Bool("debug", false, "Enable debug logging")
		dryRun      = flag.Bool("dry-run", false, "Validate config without launching")
		terminal    = flag.String("terminal", "auto", "Terminal to drive: auto, ghostty, or rex")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("ghostmux %s\n", version)
		os.Exit(0)
	}

	configsDir := filepath.Join(os.Getenv("HOME"), ".ghostmux")

	if *listConfigs {
		listAvailableConfigs(configsDir)
		return
	}

	// Support positional argument: ghostmux <name>
	if *configName == "" && *configPath == "" && flag.NArg() > 0 {
		*configName = flag.Arg(0)
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
	term := resolveTerminal(*terminal)
	var drv driver
	var termName string
	switch term {
	case "rex":
		drv = rex.New(sessionLabel(cfgPath), *debug)
		termName = "Rex"
	case "ghostty":
		drv = ghostty.New(*debug)
		termName = "Ghostty"
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown terminal '%s' (use auto, ghostty, or rex)\n", *terminal)
		os.Exit(1)
	}

	// Launch terminal
	if *debug {
		fmt.Printf("Launching %s...\n", termName)
	}
	if err := drv.Launch(); err != nil {
		fmt.Fprintf(os.Stderr, "Error launching %s: %v\n", termName, err)
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

// resolveTerminal picks Rex when run from inside Rex, and Ghostty otherwise.
func resolveTerminal(flagValue string) string {
	if flagValue != "auto" {
		return strings.ToLower(flagValue)
	}
	if os.Getenv("REX_SESSION") != "" && rex.Available() {
		return "rex"
	}
	return "ghostty"
}

// sessionLabel names the Rex session after the config, or after the current
// directory for a project-local .ghostmux.yml.
func sessionLabel(cfgPath string) string {
	name := strings.TrimSuffix(filepath.Base(cfgPath), filepath.Ext(cfgPath))
	if name == ".ghostmux" {
		if wd, err := os.Getwd(); err == nil {
			return filepath.Base(wd)
		}
	}
	return name
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
	fmt.Printf("\nLaunch with: ghostmux <config-name>\n")
}
