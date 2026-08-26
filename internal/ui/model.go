package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/GH-Jaider/obon/internal/ops"
	"github.com/GH-Jaider/obon/internal/scan"
)

// Scanner is the subset of *scan.Scanner the UI needs.
type Scanner interface {
	Scan(ctx context.Context) (*scan.Snapshot, error)
}

// Options configures the board.
type Options struct {
	Interval time.Duration // auto-refresh period (>0)
	Agents   []string      // unused here; scanner already configured upstream
	Source   Scanner
}

// Model is the bubbletea model for the lantern board.
type Model struct {
	opts   Options
	st     styles
	src    Scanner
	trk    *scan.Tracker
	states map[string]scan.DiffState

	snap      *scan.Snapshot
	view      []*scan.Group // filtered + sorted
	cursor    int           // index into view
	offset    int           // scroll offset
	cursorKey string        // identity of the cursor row; survives rescans
	select_   map[string]bool

	filtering bool
	filter    textinput.Model
	applied   string

	sortKey scan.SortKey
	sortAsc bool

	detail       *scan.Group
	detailScroll int
	helpOpen     bool
	probe        *probeResult // what localhost:port answered, for the open detail
	probeKey     string       // group key the probe belongs to

	confirm *confirmState

	toast      string
	toastUntil time.Time

	spin spinner.Model
	busy bool

	width, height int
	errText       string
	home          string // current $HOME, for ~ abbreviation
}

type confirmState struct {
	targets []ops.Target
	safety  []scan.Safety // parallel to targets
}

// messages
type tickMsg struct{}
type scannedMsg struct {
	snap *scan.Snapshot
	err  error
}
type killDoneMsg struct{ results []ops.Result }
type toastExpireMsg struct{}

// New builds the initial model.
func New(opts Options) Model {
	fi := textinput.New()
	fi.Placeholder = "port, process, cwd, origin…"
	fi.Prompt = "filter ▸ "
	fi.CharLimit = 120
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(
		lipgloss.NewStyle().Foreground(lantern)))
	m := Model{
		opts:    opts,
		st:      newStyles(),
		src:     opts.Source,
		trk:     scan.NewTracker(2),
		select_: map[string]bool{},
		sortKey: scan.SortPort,
		sortAsc: true,
		filter:  fi,
		spin:    sp,
		busy:    true,
		states:  map[string]scan.DiffState{},
		home:    userHome(),
	}
	return m
}

// Init kicks off the first scan and the refresh ticker.
func (m Model) Init() tea.Cmd {
	return tea.Batch(scanCmd(m.src), m.tick(), m.spin.Tick)
}

func (m Model) tick() tea.Cmd {
	interval := m.opts.Interval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return tea.Tick(interval, func(time.Time) tea.Msg { return tickMsg{} })
}

func scanCmd(sc Scanner) tea.Cmd {
	return func() tea.Msg {
		if sc == nil {
			return scannedMsg{err: fmt.Errorf("nil source")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		snap, err := sc.Scan(ctx)
		return scannedMsg{snap: snap, err: err}
	}
}

// Update routes messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.busy {
			var cmd tea.Cmd
			m.spin, cmd = m.spin.Update(msg)
			return m, cmd
		}
		return m, nil

	case tickMsg:
		cmds := []tea.Cmd{m.tick()}
		if !m.busy {
			m.busy = true
			cmds = append(cmds, scanCmd(m.src))
			cmds = append(cmds, m.spin.Tick)
		}
		return m, tea.Batch(cmds...)

	case scannedMsg:
		m.busy = false
		if msg.err != nil {
			m.errText = msg.err.Error()
			return m, nil
		}
		m.errText = ""
		m.applySnapshot(msg.snap)
		return m, nil

	case killDoneMsg:
		m.confirm = nil
		cmd := m.report(msg.results)
		// immediate re-scan so freed rows drift away right away
		m.busy = true
		return m, tea.Batch(scanCmd(m.src), cmd, m.spin.Tick)

	case probedMsg:
		if m.probeKey == msg.key {
			res := msg.res
			m.probe = &res
		}
		return m, nil

	case openedMsg:
		return m, m.setToast("Opened " + msg.url)

	case toastExpireMsg:
		if !m.toastUntil.IsZero() && !time.Now().Before(m.toastUntil) {
			m.toast = ""
			m.toastUntil = time.Time{}
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.clampCursor()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) applySnapshot(snap *scan.Snapshot) {
	m.snap = snap
	groups := make([]*scan.Group, len(snap.Groups))
	copy(groups, snap.Groups)

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

	keys := make([]string, len(groups))
	for i, g := range groups {
		keys[i] = g.Key
	}
	m.states = m.trk.Update(keys)

	// drop selections whose rows are fully gone
	live := map[string]bool{}
	for _, k := range keys {
		live[k] = true
	}
	for k := range m.select_ {
		if !live[k] {
			delete(m.select_, k)
		}
	}
	m.clampCursor()
}

func (m *Model) clampCursor() {
	n := len(m.view)
	if n == 0 {
		m.cursor, m.cursorKey, m.offset = 0, "", 0
		return
	}
	// restore by identity: find where the remembered row landed
	for i, g := range m.view {
		if g.Key == m.cursorKey {
			m.cursor = i
			m.fixOffset()
			return
		}
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.cursorKey = m.view[m.cursor].Key
	m.fixOffset()
}

func (m *Model) fixOffset() {
	rows := m.bodyRows()
	if rows <= 0 {
		m.offset = 0
		return
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
	if m.offset > len(m.view)-rows {
		m.offset = len(m.view) - rows
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m *Model) report(results []ops.Result) tea.Cmd {
	if len(results) == 0 {
		return nil
	}
	okN := 0
	freePorts := []int{}
	for _, r := range results {
		if r.Err != nil {
			continue
		}
		okN++
		if r.PortFree && r.Port > 0 {
			freePorts = append(freePorts, r.Port)
		}
	}
	msg := fmt.Sprintf("Sent off %d · ", okN)
	switch {
	case len(freePorts) == 1:
		msg += fmt.Sprintf(":%d is free", freePorts[0])
	case len(freePorts) > 1:
		parts := ""
		for i, p := range freePorts {
			if i > 0 {
				parts += ", "
			}
			parts += ":" + fmt.Sprint(p)
		}
		msg += parts + " are free"
	default:
		msg += "the river carries them away"
	}
	for _, r := range results {
		if r.Err != nil {
			msg += fmt.Sprintf(" · %s (pid %d) resisted", r.Name, r.PID)
		}
	}
	return m.setToast(msg)
}

func (m *Model) setToast(s string) tea.Cmd {
	m.toast = s
	m.toastUntil = time.Now().Add(5 * time.Second)
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg { return toastExpireMsg{} })
}
