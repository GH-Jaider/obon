package scan

import (
	"context"
	"testing"
	"time"
)

// fakeSource is an in-memory Source for tests.
type fakeSource struct {
	socks []Socket
	info  map[int32]*Process
	ppids map[int32]int32
}

func (f *fakeSource) Listeners(context.Context) ([]Socket, error) { return f.socks, nil }

func (f *fakeSource) Info(pid int32) (*Process, bool) {
	p, ok := f.info[pid]
	return p, ok
}

func (f *fakeSource) PPID(pid int32) (int32, bool) {
	pp, ok := f.ppids[pid]
	return pp, ok && pp > 0
}

func testChain() *fakeSource {
	now := time.Now()
	mk := func(pid int32, name string) *Process {
		return &Process{PID: pid, Name: name, StartedAt: now.Add(-time.Hour)}
	}
	return &fakeSource{
		socks: []Socket{
			{Port: 5173, Proto: TCP, Family: "ipv4", Bind: "127.0.0.1", PID: 100},
			{Port: 5173, Proto: TCP, Family: "ipv4", Bind: "127.0.0.1", PID: 200}, // SO_REUSEPORT
			{Port: 3000, Proto: TCP, Family: "ipv4", Bind: "0.0.0.0", PID: 300},
			{Port: 5353, Proto: UDP, Family: "ipv4", Bind: "127.0.0.1", PID: 400},
		},
		info: map[int32]*Process{
			100: mk(100, "vite"),
			200: mk(200, "vite"),
			300: mk(300, "node"),
			400: mk(400, "mDNSResponder"),
			500: mk(500, "claude"),
			600: mk(600, "zsh"),
		},
		ppids: map[int32]int32{
			100: 500, // vite <- claude
			200: 600, // vite <- zsh
			500: 600,
			600: 1,
			300: 1,
		},
	}
}

func TestScanOriginThroughInterpreter(t *testing.T) {
	src := testChain()
	// claude as a node script: kernel name "node", script in argv[1]
	src.socks = append(src.socks, Socket{Port: 4000, Proto: TCP, Family: "ipv4", Bind: "127.0.0.1", PID: 800})
	src.info[800] = &Process{PID: 800, Name: "vite", StartedAt: time.Now().Add(-time.Minute)}
	src.info[700] = &Process{PID: 700, Name: "node",
		Cmdline: "node /Users/jai/.local/bin/claude --continue", StartedAt: time.Now().Add(-time.Hour)}
	src.ppids[800] = 700
	src.ppids[700] = 1
	sc := NewScanner(src, Options{})
	snap, _ := sc.Scan(context.Background())
	for _, g := range snap.Groups {
		if g.Port == 4000 {
			if got := g.PIDs[0].Origin; got != "claude" {
				t.Errorf("script agent must be detected through the interpreter, got %q", got)
			}
		}
	}
}

func TestScanGroupsSharedPort(t *testing.T) {
	sc := NewScanner(testChain(), Options{})
	snap, err := sc.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var g5173 *Group
	for _, g := range snap.Groups {
		if g.Port == 5173 {
			g5173 = g
		}
	}
	if g5173 == nil {
		t.Fatal("port 5173 missing")
	}
	if len(g5173.PIDs) != 2 {
		t.Fatalf("want 2 pids sharing :5173, got %d", len(g5173.PIDs))
	}
	if len(snap.Groups) != 3 {
		t.Fatalf("want 3 groups, got %d", len(snap.Groups))
	}
}

func TestScanOriginDetection(t *testing.T) {
	sc := NewScanner(testChain(), Options{})
	snap, _ := sc.Scan(context.Background())
	byPort := map[int]*Group{}
	for _, g := range snap.Groups {
		byPort[g.Port] = g
	}
	if got := byPort[5173].PIDs[0].Origin; got != "claude" {
		t.Errorf("vite under claude: want origin claude, got %q", got)
	}
	if got := byPort[5173].PIDs[1].Origin; got != "manual" {
		t.Errorf("vite under zsh: want manual, got %q", got)
	}
	if len(byPort[5173].PIDs[0].Parents) < 2 {
		t.Error("parent chain should reach above the direct parent")
	}
}

func TestScanCustomAgents(t *testing.T) {
	sc := NewScanner(testChain(), Options{Agents: []string{"zsh"}})
	snap, _ := sc.Scan(context.Background())
	for _, g := range snap.Groups {
		if g.Port == 5173 {
			if got := g.PIDs[1].Origin; got != "zsh" {
				t.Errorf("want zsh as custom agent origin, got %q", got)
			}
		}
	}
}

func TestMatchAgent(t *testing.T) {
	cases := []struct {
		agent, name string
		want        bool
	}{
		{"claude", "claude", true},
		{"code", "Code Helper", true},
		{"code", "code-server", true},
		{"cursor", "cursor-agent", true},
		{"codex", "codexia", false}, // not a hyphen prefix of the agent itself
		{"node", "nodemon", false},
	}
	for _, c := range cases {
		if got := matchAgent(c.agent, c.name); got != c.want {
			t.Errorf("matchAgent(%q,%q)=%v want %v", c.agent, c.name, got, c.want)
		}
	}
}

func TestParentChainCycleGuard(t *testing.T) {
	f := testChain()
	f.ppids[100] = 200
	f.ppids[200] = 100 // cycle
	sc := NewScanner(f, Options{})
	snap, _ := sc.Scan(context.Background())
	for _, g := range snap.Groups {
		if g.Port != 5173 {
			continue
		}
		if len(g.PIDs[0].Parents) > 32 {
			t.Fatal("chain walked a cycle unbounded")
		}
	}
}

func TestBindWarningDetection(t *testing.T) {
	sc := NewScanner(testChain(), Options{})
	snap, _ := sc.Scan(context.Background())
	wildcard := map[string]bool{}
	for _, g := range snap.Groups {
		for _, b := range g.Binds {
			if b == "0.0.0.0" {
				wildcard[g.Key] = true
			}
		}
	}
	if !wildcard["tcp:3000"] {
		t.Error("tcp:3000 should carry a wildcard bind")
	}
	if wildcard["tcp:5173"] {
		t.Error("loopback-only group must not be flagged")
	}
}
