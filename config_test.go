package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigScaleValues(t *testing.T) {
	tests := []struct {
		name          string
		config        string
		wantUIScale   int
		wantTextScale int
	}{
		{"valid", "ui_scale = 160\ntext_scale = 125\n", 160, 125},
		{"explicit 100", "ui_scale = 100\ntext_scale = 100\n", 100, 100},
		{"minimum", "ui_scale = 80\ntext_scale = 80\n", 80, 80},
		{"maximum", "ui_scale = 200\ntext_scale = 150\n", 200, 150},
		{"too small", "ui_scale = 79\ntext_scale = 79\n", 0, 100},
		{"too large", "ui_scale = 201\ntext_scale = 151\n", 0, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(path, []byte(tt.config), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := loadConfig(path)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.UIScale != tt.wantUIScale || cfg.TextScale != tt.wantTextScale {
				t.Fatalf("scales = (%d, %d), want (%d, %d)", cfg.UIScale, cfg.TextScale, tt.wantUIScale, tt.wantTextScale)
			}
		})
	}
}

func TestLoadConfigMissingUsesScaleDefaults(t *testing.T) {
	cfg, err := loadConfig(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.UIScale != 0 || cfg.TextScale != 100 {
		t.Fatalf("default scales = (%d, %d), want (0, 100)", cfg.UIScale, cfg.TextScale)
	}
}
