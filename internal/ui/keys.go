package ui

import (
	"context"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/GH-Jaider/obon/internal/ops"
	"github.com/GH-Jaider/obon/internal/scan"
)

var sortCycle = []scan.SortKey{
	scan.SortPort,
	scan.SortSafety,
	scan.SortUptime,
	scan.SortProcess,
	scan.SortOrigin,
	scan.SortPID,
	scan.SortProto,
}

// handleKey routes keys by active layer: confirm modal, filter prompt,
// detail panel, then the table itself.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.confirm != nil {
		switch key {
		case "y", "Y", "enter":
			return m.startSendOff()
		case "n", "N", "esc", "q":
			m.confirm = nil
		}
		return m, nil
	}

	if m.filtering {
		switch key {
		case "enter":
			m.applied = m.filter.Value()
			m.filtering = false
			m.filter.Blur()
			m.rescanSoft()
			return m, scanCmd(m.src)
		case "esc":
			m.filter.SetValue(m.applied)
			m.filtering = false
			m.filter.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		m.rebuildView() // live preview while typing
		return m, cmd
	}

	if m.detail != nil {
		switch key {
		case "esc", "q", "left", "enter":
			m.detail = nil
			m.detailScroll = 0
			m.probe, m.probeKey = nil, ""
		case "o":
			if m.detail.Proto == scan.TCP {
				return m, openBrowserCmd(m.detail.Port)
			}
		case "j", "down":
			m.detailScroll++
		case "k", "up":
			if m.detailScroll > 0 {
				m.detailScroll--
			}
		case "g", "home":
			m.detailScroll = 0
		}
		return m, nil
	}

	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "?":
		m.helpOpen = !m.helpOpen

	case "j", "down":
		m.moveCursor(1)
	case "k", "up":
		m.moveCursor(-1)
	case "g", "home":
		m.setCursor(0)
	case "G", "end":
		m.setCursor(len(m.view) - 1)
	case "pgup":
		m.moveCursor(-m.bodyRows())
	case "pgdown":
		m.moveCursor(m.bodyRows())

	case "enter":
		if len(m.view) > 0 {
			g := m.view[m.cursor]
			m.detail = g
			m.detailScroll = 0
			m.probe, m.probeKey = nil, ""
			if m.showURL(g) {
				m.probeKey = g.Key
				return m, probeCmd(g.Key, g.Port)
			}
		}

	case "o":
		if len(m.view) > 0 && m.view[m.cursor].Proto == scan.TCP {
			return m, openBrowserCmd(m.view[m.cursor].Port)
		}

	case " ":
		if len(m.view) > 0 {
			k := m.view[m.cursor].Key
			if m.select_[k] {
				delete(m.select_, k)
			} else {
				m.select_[k] = true
			}
			m.moveCursor(1)
		}

	case "a":
		for _, g := range m.view {
			m.select_[g.Key] = true
		}
	case "A":
		m.select_ = map[string]bool{}

	case "x", "X", "d":
		if targets, safety := m.chosenTargets(); len(targets) > 0 {
			m.confirm = &confirmState{targets: targets, safety: safety}
		}

	case "/":
		m.filtering = true
		m.filter.Focus()
		m.filter.SetValue(m.applied)
		m.filter.CursorEnd()

	case "s":
		m.cycleSort(true)
	case "S":
		m.sortAsc = !m.sortAsc
		m.rebuildView()

	case "r":
		if !m.busy && m.src != nil {
			m.busy = true
			return m, tea.Batch(scanCmd(m.src), m.spin.Tick)
		}

	case "esc":
		switch {
		case len(m.select_) > 0:
			m.select_ = map[string]bool{}
		case m.applied != "":
			m.applied = ""
			m.filter.SetValue("")
			m.rebuildView()
		}
	}
	return m, nil
}

func (m *Model) moveCursor(delta int) {
	m.setCursor(m.cursor + delta)
}

// setCursor moves to an absolute index, remembering the row's identity.
func (m *Model) setCursor(idx int) {
	n := len(m.view)
	if n == 0 {
		m.cursor, m.cursorKey = 0, ""
		return
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	m.cursor = idx
	m.cursorKey = m.view[idx].Key
	m.fixOffset()
}

func (m *Model) cycleSort(next bool) {
	cur := 0
	for i, k := range sortCycle {
		if k == m.sortKey {
			cur = i
			break
		}
	}
	if next {
		cur = (cur + 1) % len(sortCycle)
	}
	m.sortKey = sortCycle[cur]
	m.rebuildView()
}

func (m *Model) rebuildView() {
	if m.snap == nil {
		return
	}
	groups := make([]*scan.Group, len(m.snap.Groups))
	copy(groups, m.snap.Groups)
	query := m.applied
	if m.filtering {
		query = m.filter.Value()
	}
	if query != "" {
		f := scan.CompileFilter(query)
		var kept []*scan.Group
		for _, g := range groups {
			if f.Match(g) {
				kept = append(kept, g)
			}
		}
		groups = kept
	}
	scan.SortGroups(groups, m.sortKey, m.sortAsc)
	m.view = groups
	m.clampCursor()
}

func (m *Model) rescanSoft() { m.rebuildView() }

// chosenTargets: selection if any, else the cursor row's PIDs.
// The parallel safety slice feeds the confirm dialog's verdicts.
func (m *Model) chosenTargets() ([]ops.Target, []scan.Safety) {
	var gs []*scan.Group
	if len(m.select_) > 0 {
		for _, g := range m.view {
			if m.select_[g.Key] {
				gs = append(gs, g)
			}
		}
	} else if len(m.view) > 0 {
		gs = []*scan.Group{m.view[m.cursor]}
	}
	var ts []ops.Target
	var sf []scan.Safety
	for _, g := range gs {
		for _, d := range g.PIDs {
			ts = append(ts, ops.Target{PID: d.Socket.PID, Name: d.Process.Name, Port: g.Port})
			sf = append(sf, d.Safety)
		}
	}
	return ts, sf
}

// startSendOff launches the async send-off for whatever is confirmed.
func (m Model) startSendOff() (tea.Model, tea.Cmd) {
	targets := append([]ops.Target(nil), m.confirm.targets...)
	m.confirm = nil
	m.setToast("Sending off " + pluralTargets(len(targets)) + "…")
	return m, tea.Batch(sendOffCmd(targets, 1500*time.Millisecond), m.spin.Tick)
}

func sendOffCmd(targets []ops.Target, grace time.Duration) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), grace+3*time.Second)
		defer cancel()
		res := ops.SendOff(ctx, targets, grace)
		return killDoneMsg{results: res}
	}
}

func pluralTargets(n int) string {
	if n == 1 {
		return "1 process"
	}
	return strconv.Itoa(n) + " processes"
}
