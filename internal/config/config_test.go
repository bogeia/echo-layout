package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	sample, err := os.ReadFile("../../configs/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	valid := string(sample)
	t.Setenv("SERVER_ADDRESS", "0.0.0.0:9999")
	for _, test := range []struct {
		name, content string
		valid         bool
	}{
		{"valid", valid, true},
		{"all interfaces", strings.ReplaceAll(valid, "127.0.0.1:8080", ":8080"), true},
		{"hostname", strings.ReplaceAll(valid, "127.0.0.1:8080", "localhost:8080"), true},
		{"IPv6", strings.ReplaceAll(valid, "127.0.0.1:8080", "'[::1]:8080'"), true},
		{"explicit zero", strings.ReplaceAll(valid, "max_backups: 7", "max_backups: 0"), true},
		{"console enabled", strings.ReplaceAll(valid, "console: false", "console: true"), true},
		{"explicit false", strings.ReplaceAll(valid, "compress: true", "compress: false"), true},
		{"missing console", strings.ReplaceAll(valid, "  console: false\n", ""), false},
		{"missing compress", strings.ReplaceAll(valid, "  compress: true\n", ""), false},
		{"null compress", strings.ReplaceAll(valid, "compress: true", "compress: null"), false},
		{"missing section", strings.Split(valid, "\nlog:")[0], false},
		{"unknown", valid + "unknown: true\n", true},
		{"unknown nested", valid + "  format: json\n", true},
		{"wrong bool", strings.ReplaceAll(valid, "compress: true", "compress: invalid"), false},
		{"quoted integer", strings.ReplaceAll(valid, "max_size: 500", "max_size: \"500\""), false},
		{"fractional integer", strings.ReplaceAll(valid, "max_size: 500", "max_size: 1.5"), false},
		{"duration without units", strings.ReplaceAll(valid, "read_timeout: 10s", "read_timeout: 10"), false},
		{"bad duration", strings.ReplaceAll(valid, "read_timeout: 10s", "read_timeout: nonsense"), false},
		{"zero duration", strings.ReplaceAll(valid, "read_timeout: 10s", "read_timeout: 0s"), false},
		{"bad level", strings.ReplaceAll(valid, "level: info", "level: invalid"), false},
		{"numeric level", strings.ReplaceAll(valid, "level: info", "level: 0"), false},
		{"empty filename", strings.ReplaceAll(valid, "filename: logs/app.log", "filename: ''"), false},
		{"zero size", strings.ReplaceAll(valid, "max_size: 500", "max_size: 0"), false},
		{"bad port", strings.ReplaceAll(valid, "127.0.0.1:8080", "127.0.0.1:0"), false},
		{"missing port", strings.ReplaceAll(valid, "127.0.0.1:8080", "localhost"), false},
		{"broken yaml", "server: [", false},
		{"empty", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(test.content), 0o600); err != nil {
				t.Fatal(err)
			}
			config, err := Load(path)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v, got error %v", test.valid, err)
			}
			if test.valid && !strings.Contains(test.content, config.Server.Address) {
				t.Fatalf("environment overrode configuration: %s", config.Server.Address)
			}
		})
	}
	if _, err := Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("missing file accepted")
	}
	if _, err := Load("config.json"); err == nil {
		t.Fatal("non-YAML file accepted")
	}
}
