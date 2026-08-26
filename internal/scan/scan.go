// Package scan collects everything obon knows about processes listening
// on local ports. It has no UI dependencies and is fully testable with
// fake data via the Source interface.
package scan

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Proto constants for socket protocols.
const (
	TCP = "tcp"
	UDP = "udp"
)

// Socket is one listening endpoint bound by a process.
type Socket struct {
	Port   int    `json:"port"`
	Proto  string `json:"proto"`  // tcp | udp
	Family string `json:"family"` // ipv4 | ipv6
	Bind   string `json:"bind"`   // "0.0.0.0", "127.0.0.1", "[::]", "[::1]", ...
	PID    int32  `json:"pid"`    // 0 if unresolvable without privileges
}

// ProcRef is a node in a parent chain.
type ProcRef struct {
	PID  int32  `json:"pid"`
	Name string `json:"name"`
}

// Process is the process-level detail behind a socket.
type Process struct {
	PID       int32     `json:"pid"`
	Name      string    `json:"name"`
	User      string    `json:"user,omitempty"`
	Cmdline   string    `json:"cmdline"`
	Cwd       string    `json:"cwd,omitempty"`
	StartedAt time.Time `json:"started_at"`
	PPID      int32     `json:"ppid"`
}

// Group is one row: a port/protocol pair and every PID holding it.
// Several PIDs can share a port (SO_REUSEPORT, preforking servers).
type Group struct {
	Port     int       `json:"port"`
	Proto    string    `json:"proto"`
	Binds    []string  `json:"binds"`
	PIDs     []*Detail `json:"pids"`
	FirstPID *Detail   `json:"-"`
	Key      string    `json:"-"` // "tcp:5173"
}

// Detail couples a socket with its owning process info.
type Detail struct {
	Socket    Socket        `json:"-"`
	Process   Process       `json:"-"`
	Parents   []ProcRef     `json:"parent_chain,omitempty"` // nearest ancestor first, up to init/launchd
	Origin    string        `json:"origin"`                 // agent name or "manual"
	OriginPID int32         `json:"origin_pid,omitempty"`
	Uptime    time.Duration `json:"-"`
}

// Snapshot is one full scan pass.
type Snapshot struct {
	At       time.Time `json:"at"`
	Groups   []*Group  `json:"groups"`
	Warnings []string  `json:"warnings,omitempty"`
}

// Source abstracts OS-specific enumeration so tests can inject fakes.
type Source interface {
	// Listeners returns every listening socket (TCP LISTEN + bound UDP),
	// IPv4 and IPv6.
	Listeners(ctx context.Context) ([]Socket, error)
	// Info returns process details; ok=false when the process vanished
	// or is not accessible to the current user.
	Info(pid int32) (*Process, bool)
	// PPID returns the parent pid of pid.
	PPID(pid int32) (int32, bool)
}

// DefaultAgents are the ancestor process names recognised as "origins".
var DefaultAgents = []string{
	"claude", "codex", "cursor", "cursor-agent", "code", "aider",
	"gemini", "windsurf", "copilot", "amp", "opencode", "cline",
	"tmux", "screen", "ssh",
}

// Options tunes a Scanner.
type Options struct {
	Agents  []string      // origin detection; nil = DefaultAgents
	Timeout time.Duration // per-scan budget; 0 = 5s
}

// Scanner builds Snapshots from a Source.
type Scanner struct {
	src    Source
	opts   Options
	pcache map[int32]*Process
}

// NewScanner returns a Scanner over src.
func NewScanner(src Source, opts Options) *Scanner {
	if len(opts.Agents) == 0 {
		opts.Agents = DefaultAgents
	}
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Second
	}
	return &Scanner{src: src, opts: opts}
}

// Scan produces one snapshot: sockets, joined with process detail,
// grouped by port/proto, each carrying its parent chain and origin.
func (s *Scanner) Scan(ctx context.Context) (*Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, s.opts.Timeout)
	defer cancel()

	socks, err := s.src.Listeners(ctx)
	if err != nil {
		return nil, err
	}

	s.pcache = map[int32]*Process{}
	now := time.Now()
	byKey := map[string]*Group{}
	var warns []string

	for _, sk := range socks {
		if sk.Port <= 0 || sk.Port > 65535 {
			continue
		}
		d := &Detail{Socket: sk, Origin: "manual"}
		if p, ok := s.src.Info(sk.PID); ok {
			d.Process = *p
		} else {
			d.Process = Process{PID: sk.PID, Name: unknownName(sk.PID)}
			warns = append(warns, fmt.Sprintf("pid %d: details unavailable (permissions?)", sk.PID))
		}
		s.fillChain(d)
		key := sk.Proto + ":" + itoa(sk.Port)
		g := byKey[key]
		if g == nil {
			g = &Group{Port: sk.Port, Proto: sk.Proto, Key: key}
			byKey[key] = g
		}
		g.PIDs = append(g.PIDs, d)
		g.FirstPID = g.PIDs[0]
		if !containsFold(g.Binds, bindLabel(sk)) {
			g.Binds = append(g.Binds, bindLabel(sk))
		}
	}

	groups := make([]*Group, 0, len(byKey))
	for _, g := range byKey {
		for _, d := range g.PIDs {
			d.Uptime = now.Sub(d.Process.StartedAt)
			if d.Uptime < 0 {
				d.Uptime = 0
			}
		}
		groups = append(groups, g)
	}
	sortGroups(groups)

	snap := &Snapshot{At: now, Groups: groups, Warnings: dedupe(warns)}
	return snap, nil
}

func (s *Scanner) fillChain(d *Detail) {
	pid := d.Socket.PID
	seen := map[int32]bool{}
	chain := make([]ProcRef, 0, 8)
	cur := pid
	for depth := 0; depth < 32; depth++ {
		ppid, ok := s.src.PPID(cur)
		if !ok || ppid <= 0 || ppid == cur || seen[ppid] {
			break
		}
		seen[ppid] = true
		name := ""
		if p, ok := s.src.Info(ppid); ok {
			name = p.Name
		} else {
			name = fmt.Sprintf("pid:%d", ppid)
		}
		chain = append(chain, ProcRef{PID: ppid, Name: name})
		cur = ppid
		if ppid <= 1 { // launchd / init / systemd
			break
		}
	}
	d.Parents = chain
	d.Origin, d.OriginPID = detectOrigin(s.opts.Agents, chain)
}

// detectOrigin returns the nearest matching ancestor per agents list.
func detectOrigin(agents []string, chain []ProcRef) (string, int32) {
	for _, ref := range chain {
		for _, a := range agents {
			if matchAgent(a, ref.Name) {
				return a, ref.PID
			}
		}
	}
	return "manual", 0
}

// matchAgent matches base names: "code" matches "Code" and "Code Helper";
// exact or hyphen/underscore-separated prefix on lowercased name.
func matchAgent(agent, procName string) bool {
	n := strings.ToLower(baseName(procName))
	a := strings.ToLower(agent)
	if n == a {
		return true
	}
	for _, sep := range []string{"-", "_", " "} {
		if strings.HasPrefix(n, a+sep) {
			return true
		}
	}
	return false
}

func baseName(name string) string {
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	// strip macOS helper suffixes like "-Helper (Renderer)"
	if i := strings.IndexAny(name, "."); i >= 0 && strings.Contains(strings.ToLower(name), ".app") {
		name = name[:i]
	}
	return name
}

func bindLabel(sk Socket) string {
	if sk.Family == "ipv6" {
		return "[" + sk.Bind + "]"
	}
	return sk.Bind
}

// sortGroups: port asc, then proto.
func sortGroups(gs []*Group) {
	sort.Slice(gs, func(i, j int) bool {
		if gs[i].Port != gs[j].Port {
			return gs[i].Port < gs[j].Port
		}
		return gs[i].Proto < gs[j].Proto
	})
}

func containsFold(xs []string, v string) bool {
	for _, x := range xs {
		if strings.EqualFold(x, v) {
			return true
		}
	}
	return false
}

func dedupe(xs []string) []string {
	out := xs[:0]
	seen := map[string]bool{}
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }

func unknownName(pid int32) string {
	if pid == 0 {
		return "?"
	}
	return fmt.Sprintf("pid %d", pid)
}
