package ui

import "testing"

func TestColumnsCollapseAt80(t *testing.T) {
	m := Model{width: 80}
	names := map[string]bool{}
	for _, c := range m.columns() {
		names[c.name] = true
	}
	for _, want := range []string{"PORT", "PROTO", "PROCESS", "PID", "ORIGIN", "UPTIME", "CWD"} {
		if !names[want] {
			t.Errorf("at 80 cols want column %s", want)
		}
	}
}

func TestColumnsCollapseNarrow(t *testing.T) {
	m := Model{width: 60}
	var names []string
	total := gutterW + 1
	cols := m.columns()
	for _, c := range cols {
		names = append(names, c.name)
		total += c.w + colGap
	}
	if total-1 > 60 {
		t.Errorf("columns overflow 62-col terminal: %d wide (%v)", total, names)
	}
	has := func(n string) bool {
		for _, x := range names {
			if x == n {
				return true
			}
		}
		return false
	}
	if has("CWD") {
		t.Error("CWD must collapse first on narrow terminals")
	}
	if !has("PROCESS") || !has("PID") {
		t.Error("core columns must survive")
	}
}

func TestTruncRunes(t *testing.T) {
	if got := truncRunes("hello", 4); got != "hel…" {
		t.Errorf("got %q", got)
	}
	if truncRunes("hi", 5) != "hi" {
		t.Error("short strings untouched")
	}
}
