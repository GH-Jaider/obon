package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/GH-Jaider/obon/internal/scan"
)

// colSpec is one visible table column.
type colSpec struct {
	name string
	w    int
	key  scan.SortKey // -1 for non-sortable
}

const (
	gutterW  = 2 // cursor ▸ / selection ● / spacer ·
	colGap   = 1
	minProcW = 12
	minCwdW  = 18
)

// columns decides which columns survive at this width. Low-priority
// columns collapse instead of wrapping: CWD first, then Uptime, then Origin.
func (m Model) columns() []colSpec {
	w := m.width
	if w <= 0 {
		w = 80
	}
	cols := []colSpec{
		{"PORT", 5, scan.SortPort},
		{"PROTO", 7, scan.SortProto},
		{"PROCESS", minProcW, scan.SortProcess},
		{"PID", 7, scan.SortPID},
	}
	spent := gutterW + colGap
	for _, c := range cols {
		spent += c.w + colGap
	}
	if w >= spent+9+colGap {
		cols = append(cols, colSpec{"ORIGIN", 9, scan.SortOrigin})
		spent += 9 + colGap
	}
	if w >= spent+8+colGap {
		cols = append(cols, colSpec{"UPTIME", 8, scan.SortUptime})
		spent += 8 + colGap
	}
	if rest := w - spent - minCwdW; rest >= 0 {
		flex := minProcW + rest
		for i := range cols {
			if cols[i].name == "PROCESS" {
				cols[i].w = flex
			}
		}
		cols = append(cols, colSpec{"CWD", minCwdW + maxInt(0, w-spent-minCwdW-flex), -1})
	} else {
		for i := range cols {
			if cols[i].name == "PROCESS" {
				cols[i].w = maxInt(minProcW, w-spent+minProcW)
			}
		}
	}
	return cols
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// bodyRows is how many table lines fit between header and footer.
// Single source of truth for layout math (scroll, paging, rendering).
func (m Model) bodyRows() int {
	return m.height - 5 // title + col headers + blank gap + 2 footer lines
}

// View renders the whole board.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "obon"
	}
	header := m.renderHeader()
	bodyRows := m.bodyRows()

	var body string
	switch {
	case m.confirm != nil:
		body = m.centerBox(m.renderConfirm(), bodyRows)
	case m.detail != nil:
		body = m.centerBox(m.renderDetail(), bodyRows)
	case m.helpOpen:
		body = m.centerBox(m.renderHelp(), bodyRows)
	default:
		body = strings.Join(m.tableBody(bodyRows), "\n")
	}
	// hold the frame at fixed height so the footer never jumps or scrolls
	if n := strings.Count(body, "\n"); n < bodyRows-1 {
		body += strings.Repeat("\n", bodyRows-1-n)
	}
	return header + "\n\n" + body + "\n" + m.renderFooter()
}

func (m Model) tableBody(rows int) []string {
	if len(m.view) == 0 {
		return []string{m.renderEmptyState()}
	}
	out := make([]string, 0, rows)
	end := minInt(m.offset+rows, len(m.view))
	for i := m.offset; i < end; i++ {
		out = append(out, m.renderRow(i))
	}
	return out
}

// centerBox centers a finished block vertically and horizontally.
func (m Model) centerBox(box string, rows int) string {
	lines := strings.Split(box, "\n")
	padTop := (rows - len(lines)) / 2
	if padTop < 0 {
		padTop = 0
	}
	out := make([]string, 0, len(lines)+padTop)
	for i := 0; i < padTop; i++ {
		out = append(out, "")
	}
	for _, l := range lines {
		out = append(out, lipgloss.PlaceHorizontal(m.width, lipgloss.Center, l))
	}
	return strings.Join(out, "\n")
}

func (m Model) renderHeader() string {
	left := m.st.title.Render("obon")
	right := m.busyIndicator() + m.summaryText()
	titleLine := left
	if pad := m.width - lipgloss.Width(left) - lipgloss.Width(right); pad > 0 && right != "" {
		titleLine += strings.Repeat(" ", pad) + right
	}

	var hb strings.Builder
	hb.WriteString(strings.Repeat(" ", gutterW))
	for i, c := range m.columns() {
		if i > 0 {
			hb.WriteString(strings.Repeat(" ", colGap))
		}
		name := c.name
		sty := m.st.header
		if c.key >= 0 && c.key == m.sortKey {
			sty = m.st.headerSort
			if m.sortAsc {
				name += " ↑"
			} else {
				name += " ↓"
			}
		}
		hb.WriteString(sty.Render(padRight(name, c.w)))
	}
	return titleLine + "\n" + hb.String()
}

func (m Model) busyIndicator() string {
	if !m.busy {
		return ""
	}
	return m.spin.View() + " "
}

func (m Model) summaryText() string {
	if m.snap == nil {
		return m.st.subtitle.Render("")
	}
	total := len(m.snap.Groups)
	agents := 0
	for _, g := range m.snap.Groups {
		if g.PIDs[0].Origin != "manual" {
			agents++
		}
	}
	s := fmt.Sprintf("%d spirit%s on the river", total, plural(total))
	if agents > 0 {
		s += fmt.Sprintf(" · %d lit by agents", agents)
	}
	return m.st.subtitle.Render(s)
}

func (m Model) renderRow(i int) string {
	g := m.view[i]
	d := g.PIDs[0]
	state := m.states[g.Key]
	isCursor := i == m.cursor
	isSel := m.select_[g.Key]

	// gutter: cursor arrow + selection dot
	gCursor := " "
	gSel := "·"
	if isSel {
		gSel = "●"
	}
	if isCursor {
		gCursor = "▸"
	}
	cursorSty := lipgloss.NewStyle().Foreground(dim)
	if isCursor {
		cursorSty = lipgloss.NewStyle().Foreground(accent).Bold(true)
	}
	selSty := lipgloss.NewStyle().Foreground(dim).Faint(true)
	if isSel {
		selSty = lipgloss.NewStyle().Foreground(accent)
	}
	marker := cursorSty.Render(gCursor) + selSty.Render(gSel)

	line := marker
	for ci, c := range m.columns() {
		if ci > 0 {
			line += strings.Repeat(" ", colGap)
		}
		text := padRight(truncRunes(cellValue(g, c.name), c.w), c.w)

		if state == scan.Departing {
			line += m.st.rowGone.Render(text)
			continue
		}
		if state == scan.Arrived {
			line += m.st.rowFlash.Render(text)
			continue
		}
		line += m.cellStyle(g, d, c.name, isCursor).Render(text)
	}
	return line
}

func cellValue(g *scan.Group, col string) string {
	d := g.PIDs[0]
	switch col {
	case "PORT":
		return fmt.Sprint(g.Port)
	case "PROTO":
		return strings.ToUpper(g.Proto) + protoMark(g.Binds)
	case "PROCESS":
		return d.Process.Name + extraProcs(g)
	case "PID":
		return fmt.Sprint(d.Socket.PID)
	case "ORIGIN":
		return d.Origin
	case "UPTIME":
		return humanDur(d.Uptime)
	case "CWD":
		return d.Process.Cwd
	}
	return ""
}

func (m Model) cellStyle(g *scan.Group, d *scan.Detail, col string, isCursor bool) lipgloss.Style {
	isAgent := d.Origin != "manual"
	switch col {
	case "PORT":
		if !isCursor {
			return m.st.row.Foreground(dim)
		}
		return m.st.row.Bold(true)
	case "PROTO":
		if protoMark(g.Binds) == "*" {
			return m.st.warn
		}
		return m.st.dimText
	case "PROCESS":
		st := m.st.row
		if isAgent {
			st = m.st.rowAgent
		}
		if isCursor {
			st = st.Bold(true)
		}
		return st
	case "PID":
		return m.st.dimText
	case "ORIGIN":
		if isAgent {
			return m.st.originAg
		}
		return m.st.originMan
	case "UPTIME":
		return m.st.dimText
	case "CWD":
		return m.st.dimText.Faint(true)
	}
	return m.st.row
}

func protoMark(binds []string) string {
	for _, b := range binds {
		if b == "0.0.0.0" || b == "::" || b == "[::]" {
			return "*" // exposed on every interface
		}
	}
	return ""
}

func extraProcs(g *scan.Group) string {
	if n := len(g.PIDs); n > 1 {
		return fmt.Sprintf(" +%d", n-1)
	}
	return ""
}

func (m Model) renderEmptyState() string {
	center := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center)
	switch {
	case m.filtering:
		return center.Render(m.st.dimText.Render(""))
	case m.applied != "":
		return center.Render(m.st.dimText.Render(
			fmt.Sprintf("Nothing matches %q — esc clears the filter", m.applied)))
	case m.errText != "":
		return center.Render(m.st.warn.Render("! " + m.errText))
	case m.busy && m.snap == nil:
		return center.Render(m.st.dimText.Render(m.spin.View() + " lighting the lanterns…"))
	default:
		return center.Render(strings.Join([]string{
			m.st.empty.Render("No spirits lingering. The river is clear."),
			m.st.dimText.Render("r rescan · q quit"),
		}, "\n"))
	}
}

func (m Model) renderFooter() string {
	return m.contextLine() + "\n" + m.hintLine()
}

func (m Model) contextLine() string {
	if m.toast != "" {
		return m.st.toast.Render(truncRunes(m.toast, m.width-1))
	}
	var parts []string
	if m.filtering {
		parts = append(parts, m.st.filterBox.Render(m.filter.View()))
	} else if m.applied != "" {
		parts = append(parts, m.st.filterBox.Render("/"+m.applied))
	}
	info := fmt.Sprintf("%d shown", len(m.view))
	if selN := len(m.select_); selN > 0 {
		info += fmt.Sprintf(" · %d selected", selN)
	}
	if len(m.view) > 0 && m.cursor < len(m.view) {
		if cmd := m.view[m.cursor].PIDs[0].Process.Cmdline; cmd != "" {
			budget := m.width - lipgloss.Width(info) - 6
			if budget > 10 {
				info += "  " + truncRunes(cmd, budget)
			}
		}
	}
	parts = append(parts, m.st.dimText.Render(info))
	return strings.Join(parts, "  ")
}

func (m Model) hintLine() string {
	type hk struct{ k, v string }
	var keys []hk
	switch {
	case m.confirm != nil:
		keys = []hk{{"y", "send off"}, {"n/esc", "keep them"}}
	case m.filtering:
		keys = []hk{{"enter", "apply"}, {"esc", "cancel"}}
	case m.detail != nil:
		keys = []hk{{"j/k", "scroll"}, {"esc", "close"}}
	default:
		keys = []hk{
			{"j/k", "move"}, {"space", "select"}, {"x", "send off"},
			{"enter", "detail"}, {"/", "filter"}, {"s", "sort"},
			{"r", "rescan"}, {"?", "help"}, {"q", "quit"},
		}
	}
	var out strings.Builder
	for i, k := range keys {
		if i > 0 {
			out.WriteString(m.st.help.Render(" · "))
		}
		out.WriteString(m.st.helpKey.Render(k.k))
		out.WriteString(m.st.help.Render(" " + k.v))
	}
	return out.String()
}

func humanDur(d time.Duration) string {
	s := int(d.Seconds())
	switch {
	case s < 60:
		return fmt.Sprintf("%ds", s)
	case s < 3600:
		return fmt.Sprintf("%dm%02ds", s/60, s%60)
	case s < 86400:
		return fmt.Sprintf("%dh%02dm", s/3600, (s%3600)/60)
	default:
		return fmt.Sprintf("%dd%02dh", s/86400, (s%86400)/3600)
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func padRight(s string, w int) string {
	l := lipgloss.Width(s)
	if l >= w {
		return s
	}
	return s + strings.Repeat(" ", w-l)
}

func truncRunes(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 1 {
		return string(r[:maxInt(w, 0)])
	}
	return string(r[:w-1]) + "…"
}
