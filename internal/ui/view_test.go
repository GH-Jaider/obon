package ui

import "testing"

func TestColumnsAllPresentAt80(t *testing.T) {
	m := Model{width: 80}
	names := map[string]bool{}
	for _, c := range m.columns() {
		names[c.name] = true
	}
	for _, want := range []string{"PORT", "PROTO", "SAFE", "PROCESS", "PID", "ORIGIN", "UPTIME", "CWD"} {
		if !names[want] {
			t.Errorf("at 80 cols want column %s", want)
		}
	}
}

func TestColumnsFitTheirBudget(t *testing.T) {
	for _, width := range []int{60, 80, 120, 200} {
		m := Model{width: width}
		total := gutterW
		var names []string
		for _, c := range m.columns() {
			names = append(names, c.name)
			total += c.w + colGap
		}
		total -= colGap
		if total > m.contentW() {
			t.Errorf("at %d cols columns overflow: %d > %d (%v)", width, total, m.contentW(), names)
		}
	}
}

func TestColumnsCollapseNarrow(t *testing.T) {
	m := Model{width: 60}
	names := map[string]bool{}
	for _, c := range m.columns() {
		names[c.name] = true
	}
	if names["CWD"] {
		t.Error("CWD must collapse first on narrow terminals")
	}
	if !names["PROCESS"] || !names["PID"] || !names["SAFE"] {
		t.Error("core columns must survive")
	}
}

func TestTruncCell(t *testing.T) {
	if got := truncCell("hello", 4); got != "hel…" {
		t.Errorf("got %q", got)
	}
	if truncCell("hi", 5) != "hi" {
		t.Error("short strings untouched")
	}
}

func TestTruncPathKeepsTail(t *testing.T) {
	if got := truncPath("/Users/jai/Work/inalambria/app", 16); got != "…inalambria/app" && len([]rune(got)) != 16 {
		t.Errorf("want 16-rune tail, got %q", got)
	}
	if truncPath("~/x", 10) != "~/x" {
		t.Error("short paths untouched")
	}
}

func TestTildeHome(t *testing.T) {
	if got := tildeHome("/Users/jai/Projects/x", "/Users/jai"); got != "~/Projects/x" {
		t.Errorf("got %q", got)
	}
	if got := tildeHome("/Users/jaime/x", "/Users/jai"); got != "/Users/jaime/x" {
		t.Errorf("prefix collision must not abbreviate: got %q", got)
	}
}
