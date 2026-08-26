package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/GH-Jaider/obon/internal/scan"
)

// config mirrors ~/.config/obon/config.json.
type config struct {
	IntervalSecs int      `json:"interval_s"`
	Agents       []string `json:"agents"`
}

func loadConfig() config {
	cfg := config{IntervalSecs: 2}
	path := configPath()
	if path == "" {
		return cfg
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	var user config
	if json.Unmarshal(data, &user) == nil {
		if user.IntervalSecs > 0 {
			cfg.IntervalSecs = user.IntervalSecs
		}
		if len(user.Agents) > 0 {
			cfg.Agents = user.Agents
		}
	}
	return cfg
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "obon", "config.json")
}

func newScanner(cfg config) *scan.Scanner {
	return scan.NewScanner(scan.NewGopsSource(), scan.Options{Agents: cfg.Agents})
}

func scanOnce(ctx context.Context, sc *scan.Scanner) *scan.Snapshot {
	snap, err := sc.Scan(ctx)
	if err != nil {
		fatalf("obon: %v", err)
	}
	return snap
}

func fatalf(f string, args ...any) {
	fmt.Fprintf(os.Stderr, f+"\n", args...)
	os.Exit(1)
}

func humanDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		hours := int(d.Hours())
		return fmt.Sprintf("%dd%02dh", hours/24, hours%24)
	}
}
