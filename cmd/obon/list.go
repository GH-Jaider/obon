package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/GH-Jaider/obon/internal/scan"
)

func runList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "output machine-readable JSON")
	proto := fs.String("proto", "", "filter by protocol: tcp or udp")
	fs.Parse(args)

	cfg := loadConfig()
	snap := scanOnce(context.Background(), newScanner(cfg))
	groups := snap.Groups
	if *proto != "" {
		p := strings.ToLower(*proto)
		var kept []*scan.Group
		for _, g := range groups {
			if g.Proto == p {
				kept = append(kept, g)
			}
		}
		groups = kept
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(groups); err != nil {
			fatalf("obon: %v", err)
		}
		return
	}

	if len(groups) == 0 {
		fmt.Println("No spirits lingering. The river is clear.")
		return
	}

	w := struct {
		port, proto, safe, proc, pid, user, origin, up, cwd int
	}{6, 5, 7, 16, 7, 8, 9, 7, 30}
	header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %s",
		w.port, "PORT", w.proto, "PROTO", w.safe, "SAFETY", w.proc, "PROCESS", w.pid, "PID",
		w.user, "USER", w.origin, "ORIGIN", w.up, "UPTIME", "CWD")
	fmt.Println(header)
	for _, g := range groups {
		d := g.PIDs[0]
		extra := ""
		if len(g.PIDs) > 1 {
			extra = fmt.Sprintf(" (+%d)", len(g.PIDs)-1)
		}
		fmt.Printf("%-*d  %-*s  %-*s  %-*s  %-*d  %-*s  %-*s  %-*s  %s\n",
			w.port, g.Port,
			w.proto, strings.ToUpper(g.Proto)+bindMark(g),
			w.safe, d.Safety.Label,
			w.proc, trunc(d.Process.Name+extra, w.proc),
			w.pid, d.Socket.PID,
			w.user, trunc(d.Process.User, w.user),
			w.origin, trunc(d.Origin, w.origin),
			w.up, humanDuration(d.Uptime),
			trunc(d.Process.Cwd, w.cwd))
	}
}

func bindMark(g *scan.Group) string {
	for _, b := range g.Binds {
		if b == "0.0.0.0" || b == "[::]" || b == "::" {
			return "*"
		}
	}
	return ""
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
