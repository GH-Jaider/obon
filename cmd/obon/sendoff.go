package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/GH-Jaider/obon/internal/ops"
	"github.com/GH-Jaider/obon/internal/scan"
)

// resolveTargets turns CLI selectors (ports or PIDs) into ops.Target list.
func resolveTargets(snap *scan.Snapshot, selectors []string) ([]ops.Target, []string) {
	var targets []ops.Target
	var missing []string
	for _, sel := range selectors {
		n, err := strconv.Atoi(strings.TrimPrefix(sel, ":"))
		if err != nil {
			missing = append(missing, sel)
			continue
		}
		matched := false
		for _, g := range snap.Groups {
			if g.Port == n {
				for _, d := range g.PIDs {
					targets = append(targets, ops.Target{PID: d.Socket.PID, Name: d.Process.Name, Port: g.Port})
				}
				matched = true
			}
		}
		for _, g := range snap.Groups {
			for _, d := range g.PIDs {
				if int(d.Socket.PID) == n {
					targets = append(targets, ops.Target{PID: d.Socket.PID, Name: d.Process.Name, Port: g.Port})
					matched = true
				}
			}
		}
		if !matched {
			missing = append(missing, sel)
		}
	}
	return targets, missing
}

func runKill(args []string) {
	fs := flag.NewFlagSet("kill", flag.ExitOnError)
	yes := fs.Bool("y", false, "do not ask for confirmation")
	grace := fs.Duration("grace", 2*time.Second, "grace period before SIGKILL")
	fs.Parse(args)
	if fs.NArg() == 0 {
		fatalf("usage: obon kill <port|pid>...")
	}

	cfg := loadConfig()
	snap := scanOnce(context.Background(), newScanner(cfg))
	targets, missing := resolveTargets(snap, fs.Args())
	for _, m := range missing {
		fmt.Fprintf(os.Stderr, "obon: nothing listening for %q\n", m)
	}
	if len(targets) == 0 {
		os.Exit(1)
	}

	names := describeTargets(targets)
	if !*yes && !confirmSendOff(targets, names) {
		fmt.Println("Cancelled. The spirits stay.")
		return
	}
	printReport(ops.SendOff(context.Background(), targets, *grace))
}

func describeTargets(ts []ops.Target) string {
	parts := make([]string, 0, len(ts))
	for _, t := range ts {
		parts = append(parts, fmt.Sprintf("%s :%d (pid %d)", t.Name, t.Port, t.PID))
	}
	return strings.Join(parts, ", ")
}

func confirmSendOff(ts []ops.Target, names string) bool {
	fmt.Printf("Send off %d process%s? (%s)\n[y/N] ", len(ts), plural(len(ts)), names)
	var answer string
	fmt.Scanln(&answer)
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true
	}
	return false
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "es"
}

func printReport(results []ops.Result) {
	for _, r := range results {
		label := fmt.Sprintf("%s (pid %d, port %d)", r.Name, r.PID, r.Port)
		if r.Err != nil {
			fmt.Printf("  ✕ %s: %v\n", label, r.Err)
			continue
		}
		state := "the port is free"
		if !r.PortFree && r.Exited {
			state = "process gone, port state unknown"
		} else if !r.PortFree {
			state = "WARNING: port still occupied"
		}
		fmt.Printf("  ✓ sent off %s — %s\n", label, state)
	}
}

func runClean(args []string) {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	older := fs.Duration("older-than", 0, "send off listeners younger than this (e.g. 30m, 2h)")
	yes := fs.Bool("y", false, "do not ask for confirmation")
	grace := fs.Duration("grace", 2*time.Second, "grace period before SIGKILL")
	fs.Parse(args)
	if *older <= 0 {
		fatalf("usage: obon clean --older-than <duration>")
	}

	cfg := loadConfig()
	snap := scanOnce(context.Background(), newScanner(cfg))
	var targets []ops.Target
	for _, g := range snap.Groups {
		for _, d := range g.PIDs {
			if d.Uptime >= *older {
				continue
			}
			targets = append(targets, ops.Target{PID: d.Socket.PID, Name: d.Process.Name, Port: g.Port})
		}
	}
	if len(targets) == 0 {
		fmt.Println("Nothing to send off — the river is clear.")
		return
	}
	if !*yes && !confirmSendOff(targets, describeTargets(targets)) {
		fmt.Println("Cancelled. The spirits stay.")
		return
	}
	printReport(ops.SendOff(context.Background(), targets, *grace))
}
