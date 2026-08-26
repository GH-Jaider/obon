package scan

import (
	"testing"
	"time"
)

func mkgroup(port int, proto, name, cwd, origin string) *Group {
	return &Group{
		Port: port, Proto: proto,
		Binds: []string{"127.0.0.1"},
		PIDs: []*Detail{{
			Socket:  Socket{Port: port, Proto: proto, PID: int32(port)},
			Process: Process{Name: name, Cmdline: name + " --port " + itoa(port), Cwd: cwd},
			Origin:  origin,
			Uptime:  time.Duration(port) * time.Second,
		}},
	}
}

func TestFilterSubstring(t *testing.T) {
	f := CompileFilter("vite")
	if !f.Match(mkgroup(5173, TCP, "vite", "/x", "manual")) {
		t.Error("should match process name")
	}
	if f.Match(mkgroup(3000, TCP, "node", "/x", "manual")) {
		t.Error("must not match unrelated row")
	}
}

func TestFilterFields(t *testing.T) {
	cases := []struct {
		q    string
		want bool
	}{
		{"5173", true},   // port
		{"tcp", true},    // proto
		{"/srv", true},   // cwd
		{"claude", true}, // origin
		{"9999", false},  // absent port
		{"VITE", true},   // case-insensitive
	}
	g := mkgroup(5173, TCP, "vite", "/srv/app", "claude")
	for _, c := range cases {
		if got := CompileFilter(c.q).Match(g); got != c.want {
			t.Errorf("filter %q = %v, want %v", c.q, got, c.want)
		}
	}
}

func TestFilterRegex(t *testing.T) {
	f := CompileFilter("^vi.*e$")
	if !f.Match(mkgroup(1, TCP, "vite", "", "")) {
		t.Error("regex should match vite")
	}
	bad := CompileFilter("([unclosed")
	if bad.re != nil {
		t.Error("invalid regex must fall back to substring")
	}
	if !bad.Match(mkgroup(1, TCP, "x([unclosed", "", "")) {
		t.Error("fallback substring should match literal")
	}
}

func TestSortGroups(t *testing.T) {
	gs := []*Group{
		mkgroup(3000, TCP, "node", "", ""),
		mkgroup(80, TCP, "nginx", "", ""),
		mkgroup(5173, UDP, "dns", "", ""),
	}
	SortGroups(gs, SortPort, true)
	if gs[0].Port != 80 || gs[2].Port != 5173 {
		t.Errorf("asc by port broken: %d,%d,%d", gs[0].Port, gs[1].Port, gs[2].Port)
	}
	SortGroups(gs, SortPort, false)
	if gs[0].Port != 5173 {
		t.Error("desc by port broken")
	}
	SortGroups(gs, SortProcess, true)
	if gs[0].PIDs[0].Process.Name != "dns" {
		t.Errorf("sort by process: want dns first, got %s", gs[0].PIDs[0].Process.Name)
	}
}

func TestParseSortKey(t *testing.T) {
	if k, ok := ParseSortKey("Uptime"); !ok || k != SortUptime {
		t.Error("uptime key not parsed")
	}
	if _, ok := ParseSortKey("nope"); ok {
		t.Error("unknown key must fail")
	}
}

func TestHumanDurationsInFilterStability(t *testing.T) {
	// tie-breaker: equal keys stay ordered by port
	a := mkgroup(1000, TCP, "same", "", "")
	b := mkgroup(2000, TCP, "same", "", "")
	gs := []*Group{b, a}
	SortGroups(gs, SortOrigin, true)
	if gs[0].Port != 1000 {
		t.Error("tie-break by port failed")
	}
}
