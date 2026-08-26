package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/GH-Jaider/obon/internal/scan"
)

// fakeSnap builds a little river with one of each safety verdict.
func fakeSnap() *scan.Snapshot {
	mk := func(port int, proto, name, origin, cwd string, lv scan.SafetyLevel, up time.Duration) *scan.Group {
		d := &scan.Detail{
			Socket:  scan.Socket{Port: port, Proto: proto, PID: int32(port + 40000)},
			Process: scan.Process{PID: int32(port + 40000), Name: name, Cwd: cwd, Cmdline: name + " serve"},
			Origin:  origin,
			Uptime:  up,
			Safety:  scan.Safety{Level: lv, Label: lv.String(), Reason: "r", Consequence: "c"},
		}
		return &scan.Group{Port: port, Proto: proto, Key: proto + ":" + name, PIDs: []*scan.Detail{d}}
	}
	return &scan.Snapshot{Groups: []*scan.Group{
		mk(3722, "udp", "rapportd", "manual", "/", scan.SafetySystem, 14*24*time.Hour),
		mk(5173, "tcp", "node", "claude", "/Users/jai/Projects/obon", scan.SafetySafe, 90*time.Minute),
		mk(6463, "tcp", "Discord Helper", "manual", "/", scan.SafetyCaution, 3*time.Hour),
		mk(9277, "tcp", "stable", "manual", "/Users/jai", scan.SafetyUnknown, 8*time.Hour),
	}}
}

func testModel(w, h int) Model {
	m := New(Options{Interval: time.Hour})
	m.width, m.height = w, h
	m.busy = false
	m.home = "/Users/jai"
	m.applySnapshot(fakeSnap())
	return m
}

// TestDetailScrollsInsideFrame gives the detail a command long enough
// to overflow a short terminal and checks the box never breaks: line
// count within the body region and the bottom border still there.
func TestDetailScrollsInsideFrame(t *testing.T) {
	m := testModel(100, 20)
	long := strings.Repeat("/very/long/path/segment", 12) + "/server.ts --flag value"
	m.view[0].PIDs[0].Process.Cmdline = long
	m.detail = m.view[0]

	box := m.renderDetail()
	lines := strings.Split(box, "\n")
	if len(lines) > m.bodyRows() {
		t.Fatalf("detail box is %d lines, body fits %d", len(lines), m.bodyRows())
	}
	if !strings.Contains(lines[len(lines)-1], "╰") {
		t.Error("bottom border must survive overflow")
	}
	if m.detailScrollMax() == 0 {
		t.Error("this content should need scrolling")
	}
	// scrolled to the end, the box must still fit and close
	m.detailScroll = m.detailScrollMax()
	box = m.renderDetail()
	lines = strings.Split(box, "\n")
	if len(lines) > m.bodyRows() || !strings.Contains(lines[len(lines)-1], "╰") {
		t.Error("scrolled-to-end box must stay framed")
	}
}

// TestViewNoOverflow renders the full board at several sizes and
// checks no line spills past the terminal width.
func TestViewNoOverflow(t *testing.T) {
	for _, sz := range [][2]int{{80, 24}, {100, 30}, {60, 16}, {160, 40}} {
		m := testModel(sz[0], sz[1])
		for name, frame := range map[string]string{
			"board":  m.View(),
			"help":   func() string { m.helpOpen = true; defer func() { m.helpOpen = false }(); return m.View() }(),
			"detail": func() string { m.detail = m.view[0]; defer func() { m.detail = nil }(); return m.View() }(),
			"confirm": func() string {
				ts, sf := m.chosenTargets()
				m.confirm = &confirmState{targets: ts, safety: sf}
				defer func() { m.confirm = nil }()
				return m.View()
			}(),
		} {
			for i, line := range strings.Split(frame, "\n") {
				if lw := lipgloss.Width(line); lw > sz[0] {
					t.Errorf("%dx%d %s: line %d is %d wide: %q", sz[0], sz[1], name, i, lw, line)
				}
			}
			if lines := strings.Count(frame, "\n") + 1; lines != sz[1] {
				t.Errorf("%dx%d %s: frame is %d lines, terminal is %d", sz[0], sz[1], name, lines, sz[1])
			}
		}
	}
}
