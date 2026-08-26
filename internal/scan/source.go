package scan

import (
	"context"
	"fmt"
	netip "net"
	"strings"
	"sync"
	"syscall"
	"time"

	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// GopsSource implements Source using shirou/gopsutil.
// On Linux and Windows this reads /proc and the system socket tables
// natively; on macOS gopsutil itself delegates to lsof (see README).
type GopsSource struct {
	mu    sync.Mutex
	procs map[int32]*Process
	ppids map[int32]int32
}

// NewGopsSource returns a Source backed by gopsutil with per-instance caches.
func NewGopsSource() *GopsSource {
	return &GopsSource{
		procs: map[int32]*Process{},
		ppids: map[int32]int32{},
	}
}

// Listeners returns every listening socket: TCP in LISTEN state plus
// every bound UDP socket, IPv4 and IPv6, de-duplicated.
func (g *GopsSource) Listeners(ctx context.Context) ([]Socket, error) {
	conns, err := gnet.ConnectionsWithContext(ctx, "all")
	if err != nil {
		return nil, fmt.Errorf("socket enumeration failed: %w", err)
	}
	out := make([]Socket, 0, len(conns))
	seen := map[Socket]bool{}
	for _, c := range conns {
		if c.Laddr.Port == 0 {
			continue
		}
		if c.Type == syscall.SOCK_STREAM && !strings.EqualFold(c.Status, "LISTEN") {
			continue
		}
		sk := Socket{
			Port:   int(c.Laddr.Port),
			Proto:  protoLabel(c.Type),
			Family: familyLabel(c.Laddr.IP),
			Bind:   bindHost(c.Laddr.IP),
			PID:    c.Pid,
		}
		if seen[sk] {
			continue
		}
		seen[sk] = true
		out = append(out, sk)
	}
	return out, nil
}

// Info returns cached process details for pid.
func (g *GopsSource) Info(pid int32) (*Process, bool) {
	if pid <= 0 {
		return nil, false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if p, ok := g.procs[pid]; ok {
		return p, true
	}
	p := g.infoUncached(pid)
	if p == nil {
		return nil, false
	}
	g.procs[pid] = p
	return p, true
}

// PPID returns the parent pid of pid, cached; false at or above init/launchd.
func (g *GopsSource) PPID(pid int32) (int32, bool) {
	if pid <= 1 {
		return 0, false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if pp, ok := g.ppids[pid]; ok {
		return pp, pp > 0
	}
	pr, err := process.NewProcessWithContext(context.Background(), pid)
	if err != nil {
		return 0, false
	}
	pp, err := pr.PpidWithContext(context.Background())
	if err != nil {
		return 0, false
	}
	g.ppids[pid] = pp
	return pp, pp > 0
}

func (g *GopsSource) infoUncached(pid int32) *Process {
	ctx := context.Background()
	pr, err := process.NewProcessWithContext(ctx, pid)
	if err != nil {
		return nil
	}
	name, _ := pr.NameWithContext(ctx)
	cmdline, _ := pr.CmdlineWithContext(ctx)
	cwd, _ := pr.CwdWithContext(ctx)
	user, _ := pr.UsernameWithContext(ctx)
	createMs, _ := pr.CreateTimeWithContext(ctx)
	ppid, _ := pr.PpidWithContext(ctx)
	if name == "" && cmdline != "" {
		name = baseName(strings.Fields(cmdline)[0])
	}
	if name == "" {
		name = fmt.Sprintf("pid %d", pid)
	}
	g.ppids[pid] = ppid
	started := time.UnixMilli(createMs)
	if createMs <= 0 {
		started = time.Time{}
	}
	return &Process{
		PID:       pid,
		Name:      name,
		User:      user,
		Cmdline:   cmdline,
		Cwd:       cwd,
		StartedAt: started,
		PPID:      ppid,
	}
}

func protoLabel(t uint32) string {
	if t == syscall.SOCK_STREAM {
		return TCP
	}
	return UDP
}

func familyLabel(ip string) string {
	if strings.Contains(ip, ":") {
		return "ipv6"
	}
	return "ipv4"
}

func bindHost(ip string) string {
	switch ip {
	case "", "::", "0.0.0.0":
		return ip
	}
	parsed := netip.ParseIP(strings.Trim(ip, "[]"))
	if parsed != nil && parsed.IsUnspecified() {
		if strings.Contains(ip, ":") {
			return "::"
		}
		return "0.0.0.0"
	}
	return ip
}
