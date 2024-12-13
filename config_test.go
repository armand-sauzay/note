// config_test.go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestXDGPaths(t *testing.T) {
	// Save original env vars
	originalConfigHome := os.Getenv("XDG_CONFIG_HOME")
	originalDataHome := os.Getenv("XDG_DATA_HOME")
	defer func() {
		os.Setenv("XDG_CONFIG_HOME", originalConfigHome)
		os.Setenv("XDG_DATA_HOME", originalDataHome)
	}()

	homeDir, _ := os.UserHomeDir()
	tests := []struct {
		name          string
		xdgConfigHome string
		xdgDataHome   string
		wantConfig    string
		wantData      string
	}{
		{
			name:          "XDG vars set",
			xdgConfigHome: "/custom/config",
			xdgDataHome:   "/custom/data",
			wantConfig:    "/custom/config",
			wantData:      "/custom/data",
		},
		{
			name:          "XDG vars empty",
			xdgConfigHome: "",
			xdgDataHome:   "",
			wantConfig:    filepath.Join(homeDir, ".config"),
			wantData:      filepath.Join(homeDir, ".local", "share"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("XDG_CONFIG_HOME", tt.xdgConfigHome)
			os.Setenv("XDG_DATA_HOME", tt.xdgDataHome)

			if got := getConfigHome(); got != tt.wantConfig {
				t.Errorf("getConfigHome() = %v, want %v", got, tt.wantConfig)
			}
			if got := getDataHome(); got != tt.wantData {
				t.Errorf("getDataHome() = %v, want %v", got, tt.wantData)
			}
		})
	}
}
