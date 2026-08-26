package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/GH-Jaider/obon/internal/ui"
)

var version = "0.1.0"

const usageText = `obon — every dev server your agents summoned, still lingering on your ports.

usage:
  obon                 open the lantern board (TUI)
  obon list [--json] [--proto tcp|udp]
                       list listening ports once and exit
  obon kill <port|pid>...
                       send off the processes behind those ports
  obon clean --older-than <duration> [-y]
                       send off listeners younger than the duration
  obon version         print version

keys inside the TUI: press ?`

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		cfg := loadConfig()
		interval := time.Duration(cfg.IntervalSecs) * time.Second
		if interval < time.Second {
			interval = time.Second
		}
		sc := newScanner(cfg)
		m := ui.New(ui.Options{Interval: interval, Agents: cfg.Agents, Source: sc})
		if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
			fatalf("obon: %v", err)
		}
		return
	}

	switch args[0] {
	case "list":
		runList(args[1:])
	case "kill":
		runKill(args[1:])
	case "clean":
		runClean(args[1:])
	case "version", "--version", "-v":
		fmt.Println("obon", version)
	case "help", "--help", "-h":
		fmt.Println(usageText)
	default:
		fatalf("obon: unknown command %q\n\n%s", args[0], usageText)
	}
}
