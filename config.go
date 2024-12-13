package main

import (
	"fmt"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"
)

type Layout struct {
	SidebarWidth int `yaml:"sidebar_width"`
	Padding      struct {
		Horizontal int `yaml:"horizontal"`
		Vertical   int `yaml:"vertical"`
	} `yaml:"padding"`
	Heights struct {
		Header int `yaml:"header"`
		Footer int `yaml:"footer"`
		Status int `yaml:"status"`
		Help   int `yaml:"help"`
	} `yaml:"heights"`
	HeaderGap int `yaml:"header_gap"`
}

type Config struct {
	NotesDir      string `yaml:"notes_dir"`
	ArchiveDir    string `yaml:"archive_dir"`
	DefaultEditor string `yaml:"default_editor"`
	Layout        Layout `yaml:"layout"`
	Theme         struct {
		Light string `yaml:"light"`
		Dark  string `yaml:"dark"`
	} `yaml:"theme"`
}

type Dimensions struct {
	Heights struct {
		Header int
		Footer int
		Status int
	}
	Spacing struct {
		HeaderGap int
	}
}

func getConfigDir() (string, error) {
	return filepath.Join(getConfigHome(), "note"), nil
}

func getConfigHome() string {
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" && filepath.IsAbs(xdgConfig) {
		return xdgConfig
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config")
}

func getDataHome() string {
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" && filepath.IsAbs(xdgData) {
		return xdgData
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".local", "share")
}

func DefaultConfig() *Config {
	return &Config{
		NotesDir:      filepath.Join(getDataHome(), "note"),
		ArchiveDir:    filepath.Join(getDataHome(), "note", "archive"),
		DefaultEditor: "vim",
		Layout: Layout{
			SidebarWidth: 30,
			Padding: struct {
				Horizontal int `yaml:"horizontal"`
				Vertical   int `yaml:"vertical"`
			}{
				Horizontal: 2,
				Vertical:   1,
			},
			Heights: struct {
				Header int `yaml:"header"`
				Footer int `yaml:"footer"`
				Status int `yaml:"status"`
				Help   int `yaml:"help"`
			}{
				Header: 1,
				Footer: 1,
				Status: 1,
				Help:   1,
			},
			HeaderGap: 1,
		},
		Theme: struct {
			Light string `yaml:"light"`
			Dark  string `yaml:"dark"`
		}{
			Light: "default",
			Dark:  "default",
		},
	}
}

func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	// Create notes directory if it doesn't exist
	if err := os.MkdirAll(cfg.NotesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create notes directory: %v", err)
	}

	// Create archive directory if it doesn't exist
	if err := os.MkdirAll(cfg.ArchiveDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create archive directory: %v", err)
	}

	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config file
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal default config: %v", err)
		}
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return nil, fmt.Errorf("failed to write default config: %v", err)
		}
		return cfg, nil
	}

	// Read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func SaveConfig(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	configPath := filepath.Join(cfg.NotesDir, "config.yaml")
	return os.WriteFile(configPath, data, 0644)
}

func (c *Config) CalculateHeights(totalHeight int) struct {
	Content int
	Header  int
	Footer  int
} {
	header := c.Layout.Heights.Header
	status := c.Layout.Heights.Status
	help := c.Layout.Heights.Help
	footer := status + help
	content := totalHeight - header - footer - 2 - c.Layout.HeaderGap // 2 is border size

	if content < 0 {
		content = 0
	}

	return struct {
		Content int
		Header  int
		Footer  int
	}{
		Content: content,
		Header:  header,
		Footer:  footer,
	}
}

func (c *Config) GetPadding() (horizontal, vertical int) {
	return c.Layout.Padding.Horizontal, c.Layout.Padding.Vertical
}

func (c *Config) DefaultDimensions() Dimensions {
	return Dimensions{
		Heights: struct {
			Header int
			Footer int
			Status int
		}{
			Header: c.Layout.Heights.Header,
			Footer: c.Layout.Heights.Footer,
			Status: c.Layout.Heights.Status,
		},
		Spacing: struct {
			HeaderGap int
		}{
			HeaderGap: c.Layout.HeaderGap,
		},
	}
}
