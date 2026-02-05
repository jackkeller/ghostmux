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
