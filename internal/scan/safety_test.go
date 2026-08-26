package scan

import (
	"strings"
	"testing"
)

var testEnv = SafetyEnv{Home: "/Users/jai", User: "jai"}

func assess(t *testing.T, d Detail) Safety {
	t.Helper()
	return AssessSafety(&d, testEnv)
}

func TestSafetyKnownDaemon(t *testing.T) {
	s := assess(t, Detail{
		Origin:  "manual",
		Process: Process{Name: "rapportd", User: "jai", Cmdline: "/usr/libexec/rapportd"},
	})
	if s.Level != SafetySystem {
		t.Fatalf("rapportd should be system, got %v", s.Label)
	}
	if !strings.Contains(s.Consequence, "Handoff") {
		t.Errorf("want a concrete consequence, got %q", s.Consequence)
	}
}

func TestSafetySystemPath(t *testing.T) {
	s := assess(t, Detail{
		Origin:  "manual",
		Process: Process{Name: "somethingd", User: "jai", Cmdline: "/System/Library/CoreServices/somethingd"},
	})
	if s.Level != SafetySystem {
		t.Fatalf("system-path binary should be system, got %v", s.Label)
	}
}

func TestSafetyRootOwned(t *testing.T) {
	s := assess(t, Detail{
		Origin:  "manual",
		Process: Process{Name: "customd", User: "root", Cmdline: "/opt/thing/customd"},
	})
	if s.Level != SafetySystem {
		t.Fatalf("root-owned should be system, got %v", s.Label)
	}
}

func TestSafetyDocker(t *testing.T) {
	s := assess(t, Detail{
		Origin:  "manual",
		Process: Process{Name: "com.docker.backend", User: "jai", Cmdline: "/Applications/Docker.app/Contents/MacOS/com.docker.backend"},
	})
	if s.Level != SafetyCaution {
		t.Fatalf("docker backend should be caution, got %v", s.Label)
	}
	if !strings.Contains(s.Consequence, "container") {
		t.Errorf("docker consequence should mention containers, got %q", s.Consequence)
	}
}

func TestSafetyAppHelper(t *testing.T) {
	s := assess(t, Detail{
		Origin:  "manual",
		Process: Process{Name: "Discord Helper", User: "jai", Cmdline: "/Applications/Discord.app/Contents/Frameworks/Discord Helper"},
	})
	if s.Level != SafetyCaution {
		t.Fatalf("app helper should be caution, got %v", s.Label)
	}
	if !strings.Contains(s.Reason, "Discord.app") {
		t.Errorf("reason should name the app, got %q", s.Reason)
	}
}

func TestSafetyAgentLit(t *testing.T) {
	s := assess(t, Detail{
		Origin:  "claude",
		Process: Process{Name: "node", User: "jai", Cmdline: "node server.js", Cwd: "/Users/jai/Projects/x"},
	})
	if s.Level != SafetySafe {
		t.Fatalf("agent-lit dev server should be safe, got %v", s.Label)
	}
	if !strings.Contains(s.Reason, "claude") {
		t.Errorf("reason should name the agent, got %q", s.Reason)
	}
}

func TestSafetyDevServer(t *testing.T) {
	s := assess(t, Detail{
		Origin:  "manual",
		Process: Process{Name: "node", User: "jai", Cmdline: "node dev.js", Cwd: "/Users/jai/Work/site"},
	})
	if s.Level != SafetySafe {
		t.Fatalf("node in ~ should be safe, got %v", s.Label)
	}
	if !strings.Contains(s.Reason, "~/Work/site") {
		t.Errorf("reason should show the tilde cwd, got %q", s.Reason)
	}
}

func TestSafetyHomebrewPythonIsDev(t *testing.T) {
	s := assess(t, Detail{
		Origin: "manual",
		Process: Process{
			Name:    "Python",
			User:    "jai",
			Cmdline: "/opt/homebrew/Cellar/python@3.14/3.14.5/Frameworks/Python.framework/Versions/3.14/Resources/Python.app/Contents/MacOS/Python -m http.server 8947",
			Cwd:     "/tmp/demo",
		},
	})
	if s.Level != SafetySafe {
		t.Fatalf("framework python must be a dev server, got %v (%s)", s.Label, s.Reason)
	}
}

func TestSafetyUnknown(t *testing.T) {
	s := assess(t, Detail{
		Origin:  "manual",
		Process: Process{Name: "mystery", User: "jai", Cmdline: "/opt/mystery", Cwd: "/"},
	})
	if s.Level != SafetyUnknown {
		t.Fatalf("mystery process should be unknown, got %v", s.Label)
	}
}

func TestSafetySortsSafestFirst(t *testing.T) {
	mk := func(port int, lv SafetyLevel) *Group {
		return &Group{Port: port, Proto: TCP, PIDs: []*Detail{{Safety: Safety{Level: lv}}}}
	}
	gs := []*Group{mk(1, SafetySystem), mk(2, SafetySafe), mk(3, SafetyCaution)}
	SortGroups(gs, SortSafety, true)
	if gs[0].Port != 2 || gs[2].Port != 1 {
		t.Errorf("want safe first, system last; got order %d,%d,%d", gs[0].Port, gs[1].Port, gs[2].Port)
	}
}

func TestTildePath(t *testing.T) {
	if got := tildePath("/Users/jai/Projects/x", "/Users/jai"); got != "~/Projects/x" {
		t.Errorf("got %q", got)
	}
	if got := tildePath("/Users/jaime/x", "/Users/jai"); got != "/Users/jaime/x" {
		t.Errorf("prefix collision must not abbreviate: got %q", got)
	}
}
