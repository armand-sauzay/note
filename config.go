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
	ConfigDir     string `yaml:"config_dir"`
	NotesDir      string `yaml:"notes_dir"`
	ArchiveDir    string `yaml:"archive_dir"`
	Editor        string `yaml:"editor"`
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
	configDir, _ := getConfigDir()
	return &Config{
		ConfigDir:     configDir,
		NotesDir:      filepath.Join(getDataHome(), "note"),
		ArchiveDir:    filepath.Join(getDataHome(), "note", "archive"),
		Editor:        "",
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
	configDir := cfg.ConfigDir

	// Create directories if they don't exist
	dirs := []string{
		configDir,
		cfg.NotesDir,
		cfg.ArchiveDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Set the editor before creating the config file
		cfg.Editor = cfg.GetEditor()

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

	configPath := filepath.Join(cfg.ConfigDir, "config.yaml")
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

func (c *Config) GetEditor() string {
	// 1. Check config file setting
	if c.Editor != "" {
		return c.Editor
	}

	// 2. Check NOTE_EDITOR environment variable
	if noteEditor := os.Getenv("NOTE_EDITOR"); noteEditor != "" {
		return noteEditor
	}

	// 3. Check VISUAL environment variable
	if visual := os.Getenv("VISUAL"); visual != "" {
		return visual
	}

	// 4. Check EDITOR environment variable
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	// 5. Check if /usr/bin/vi exists
	if _, err := os.Stat("/usr/bin/vi"); err == nil {
		return "/usr/bin/vi"
	}

	// 6. Final fallback to ed
	return "/bin/ed"
}
