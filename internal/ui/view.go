package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/GH-Jaider/obon/internal/scan"
)

// colSpec is one visible table column.
type colSpec struct {
	name  string
	w     int
	key   scan.SortKey // -1 for non-sortable
	right bool         // right-align numeric columns
}

// Frame geometry. The whole board lives inside a rounded border; padX
// is the breathing room inside it, and the cursor bar lives in it.
const (
	padX     = 2 // inner horizontal padding between border and content
	gutterW  = 2 // selection dot column
	colGap   = 2
	minProcW = 12
	minCwdW  = 12
)

// innerW is the width between the two border runes.
func (m Model) innerW() int {
	w := m.width
	if w <= 0 {
		w = 80
	}
	return maxInt(10, w-2)
}

// contentW is the width a row background spans.
func (m Model) contentW() int {
	return m.innerW() - 2*padX
}

// columns decides which columns survive at this width. Low-priority
// columns collapse instead of wrapping: CWD first, then Uptime, then
// Origin. Leftover width goes mostly to CWD (paths earn it) and a
// little to PROCESS.
func (m Model) columns() []colSpec {
	budget := m.contentW() - gutterW
	cols := []colSpec{
		{"PORT", 6, scan.SortPort, false},
		{"PROTO", 4, scan.SortProto, false},
		{"SAFE", 5, scan.SortSafety, false},
		{"PROCESS", minProcW, scan.SortProcess, false},
		{"PID", 6, scan.SortPID, true},
	}
	spent := -colGap
	for _, c := range cols {
		spent += c.w + colGap
	}
	if budget >= spent+colGap+7 {
		cols = append(cols, colSpec{"ORIGIN", 7, scan.SortOrigin, false})
		spent += colGap + 7
	}
	if budget >= spent+colGap+6 {
		cols = append(cols, colSpec{"UPTIME", 6, scan.SortUptime, true})
		spent += colGap + 6
	}
	if rest := budget - spent - colGap - minCwdW; rest >= 0 {
		procExtra := minInt(rest/3, 16)
		for i := range cols {
			if cols[i].name == "PROCESS" {
				cols[i].w += procExtra
			}
		}
		cols = append(cols, colSpec{"CWD", minCwdW + rest - procExtra, -1, false})
	} else {
		// no room for CWD: let PROCESS soak up whatever is left
		for i := range cols {
			if cols[i].name == "PROCESS" {
				cols[i].w = maxInt(minProcW, cols[i].w+budget-spent)
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
// Frame: border, blank, title, blank, heads, rule · rule, status,
// hints, border.
func (m Model) bodyRows() int {
	return maxInt(1, m.height-10)
}

// View renders the whole board inside its rounded frame.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "obon"
	}
	bodyRows := m.bodyRows()

	var body string
	switch {
	case m.confirm != nil:
		body = m.centerBox(m.renderConfirm(), bodyRows)
	case m.detail != nil:
		body = m.centerBox(m.renderDetail(), bodyRows)
	case m.helpOpen:
		body = m.centerBox(m.renderHelp(), bodyRows)
	case len(m.view) == 0:
		body = m.centerBox(m.renderEmptyState(), bodyRows)
	default:
		body = strings.Join(m.tableBody(bodyRows), "\n")
	}

	inner := make([]string, 0, m.height-2)
	inner = append(inner, "")
	inner = append(inner, m.titleLine())
	inner = append(inner, "")
	inner = append(inner, m.headLine())
	inner = append(inner, m.ruleLine())
	bodyLines := strings.Split(body, "\n")
	if len(bodyLines) > bodyRows {
		bodyLines = bodyLines[:bodyRows] // oversized modal on a tiny terminal
	}
	inner = append(inner, bodyLines...)
	for len(inner) < m.height-5 {
		inner = append(inner, "")
	}
	inner = append(inner, m.ruleLine())
	inner = append(inner, margin()+m.contextLine())
	inner = append(inner, margin()+m.hintLine())

	return m.frame(inner)
}

// frame wraps the inner lines in the rounded border.
func (m Model) frame(inner []string) string {
	w := m.innerW()
	b := m.st.border
	var out strings.Builder
	out.WriteString(b.Render("╭" + strings.Repeat("─", w) + "╮"))
	for _, line := range inner {
		out.WriteString("\n")
		out.WriteString(b.Render("│"))
		out.WriteString(padRight(line, w))
		out.WriteString(b.Render("│"))
	}
	out.WriteString("\n")
	out.WriteString(b.Render("╰" + strings.Repeat("─", w) + "╯"))
	return out.String()
}

func (m Model) tableBody(rows int) []string {
	out := make([]string, 0, rows)
	end := minInt(m.offset+rows, len(m.view))
	for i := m.offset; i < end; i++ {
		out = append(out, m.renderRow(i))
	}
	return out
}

// centerBox centers a finished block vertically and horizontally
// within the body region.
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
		out = append(out, lipgloss.PlaceHorizontal(m.innerW(), lipgloss.Center, l))
	}
	return strings.Join(out, "\n")
}

func margin() string { return strings.Repeat(" ", padX) }

// titleLine: the lantern badge and the river report.
func (m Model) titleLine() string {
	left := m.st.titleBadge.Render(" ◉ obon ")
	right := m.busyIndicator() + m.summaryText()
	line := margin() + left
	if pad := m.contentW() - lipgloss.Width(left) - lipgloss.Width(right); pad > 0 && right != "" {
		line += strings.Repeat(" ", pad) + right
	}
	return line
}

// headLine: the column headers, sort column lit.
func (m Model) headLine() string {
	var hb strings.Builder
	hb.WriteString(margin())
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
		cell := padRight(name, c.w)
		if c.right {
			cell = padLeft(name, c.w)
		}
		hb.WriteString(sty.Render(cell))
	}
	return hb.String()
}

func (m Model) ruleLine() string {
	return margin() + m.st.rule.Render(strings.Repeat("─", m.contentW()))
}

func (m Model) busyIndicator() string {
	if !m.busy {
		return ""
	}
	return m.spin.View() + " "
}

func (m Model) summaryText() string {
	if m.snap == nil {
		return ""
	}
	total := len(m.snap.Groups)
	agents := 0
	for _, g := range m.snap.Groups {
		if g.PIDs[0].Origin != "manual" {
			agents++
		}
	}
	s := m.st.subtitle.Render(fmt.Sprintf("%d spirit%s on the river", total, plural(total)))
	if agents > 0 {
		s += m.st.subtitle.Render(" · ") +
			m.st.originAg.Render(fmt.Sprintf("%d lit by agents", agents))
	}
	return s
}

// renderRow paints one line of the river. The cursor row rides violet
// water behind a lantern bar; freshly-lit rows glow lantern orange.
func (m Model) renderRow(i int) string {
	g := m.view[i]
	d := g.PIDs[0]
	state := m.states[g.Key]
	isCursor := i == m.cursor
	isSel := m.select_[g.Key]

	// margin: cursor bar
	bar := margin()
	if isCursor {
		bar = lipgloss.NewStyle().Foreground(lantern).Bold(true).Render("▌") + " "
	}

	// background carrier: flash > cursor > plain; gaps ride the same
	// water so the bar is unbroken
	bg := func(s lipgloss.Style) lipgloss.Style {
		switch {
		case state == scan.Arrived:
			return s.Foreground(flashInk).Background(lantern)
		case isCursor:
			return s.Background(cursorBg)
		}
		return s
	}
	gap := bg(lipgloss.NewStyle()).Render(strings.Repeat(" ", colGap))

	// selection gutter
	sel := strings.Repeat(" ", gutterW)
	if isSel {
		sel = padRight("●", gutterW)
	}
	line := bar + bg(m.st.selDot).Render(sel)

	used := gutterW
	for ci, c := range m.columns() {
		if ci > 0 {
			line += gap
			used += colGap
		}
		used += c.w

		// the SAFE pill keeps its own background on every water
		if c.name == "SAFE" && state != scan.Arrived {
			pill := m.safetyPill(d.Safety.Level)
			line += pill
			if pad := c.w - lipgloss.Width(pill); pad > 0 {
				line += bg(lipgloss.NewStyle()).Render(strings.Repeat(" ", pad))
			}
			continue
		}

		raw := cellValue(g, c.name, m.home)
		text := truncCell(raw, c.w)
		if c.name == "CWD" {
			text = truncPath(raw, c.w)
		}
		if c.right {
			text = padLeft(text, c.w)
		} else {
			text = padRight(text, c.w)
		}

		switch {
		case state == scan.Departing:
			line += bg(m.st.rowGone).Render(text)
		case state == scan.Arrived:
			line += m.st.rowFlash.Render(text)
		default:
			line += bg(m.cellStyle(g, d, c.name, isCursor)).Render(text)
		}
	}
	// extend the background to the right edge
	if rest := m.contentW() - used; rest > 0 && (isCursor || state == scan.Arrived) {
		line += bg(lipgloss.NewStyle()).Render(strings.Repeat(" ", rest))
	}
	return line
}

func cellValue(g *scan.Group, col, home string) string {
	d := g.PIDs[0]
	switch col {
	case "PORT":
		return ":" + fmt.Sprint(g.Port)
	case "PROTO":
		return g.Proto + protoMark(g.Binds)
	case "SAFE":
		return safetyWord(d.Safety.Level)
	case "PROCESS":
		return d.Process.Name + extraProcs(g)
	case "PID":
		return fmt.Sprint(d.Socket.PID)
	case "ORIGIN":
		if d.Origin == "manual" {
			return "–"
		}
		return d.Origin
	case "UPTIME":
		return humanDur(d.Uptime)
	case "CWD":
		return tildeHome(d.Process.Cwd, home)
	}
	return ""
}

// safetyWord is the three-letter verdict.
func safetyWord(l scan.SafetyLevel) string {
	switch l {
	case scan.SafetySafe:
		return "dev"
	case scan.SafetyCaution:
		return "app"
	case scan.SafetySystem:
		return "sys"
	}
	return "?"
}

// safetyPill renders the verdict as a solid-color pill, 5 cells wide.
func (m Model) safetyPill(l scan.SafetyLevel) string {
	w := safetyWord(l)
	pad := " "
	if len(w) == 1 {
		pad = "  "
	}
	return m.pillStyle(l).Render(pad + w + " ")
}

func (m Model) pillStyle(l scan.SafetyLevel) lipgloss.Style {
	switch l {
	case scan.SafetySafe:
		return m.st.pillDev
	case scan.SafetyCaution:
		return m.st.pillApp
	case scan.SafetySystem:
		return m.st.pillSys
	}
	return m.st.pillUnk
}

func (m Model) safetyStyle(l scan.SafetyLevel) lipgloss.Style {
	switch l {
	case scan.SafetySafe:
		return m.st.safeOK
	case scan.SafetyCaution:
		return m.st.safeWarn
	case scan.SafetySystem:
		return m.st.safeSys
	}
	return m.st.safeUnk
}

func (m Model) cellStyle(g *scan.Group, d *scan.Detail, col string, isCursor bool) lipgloss.Style {
	isAgent := d.Origin != "manual"
	switch col {
	case "PORT":
		if isCursor {
			return lipgloss.NewStyle().Foreground(lantern).Bold(true)
		}
		return m.st.row
	case "PROTO":
		if protoMark(g.Binds) != "" {
			return m.st.warn
		}
		return m.st.faintText
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
		return m.st.faintText
	case "ORIGIN":
		if isAgent {
			return m.st.originAg
		}
		return lipgloss.NewStyle().Foreground(faint)
	case "UPTIME":
		return m.st.dimText
	case "CWD":
		return m.st.dimText
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
	switch {
	case m.filtering:
		return ""
	case m.applied != "":
		return m.st.dimText.Render(
			fmt.Sprintf("Nothing matches %q · esc clears the filter", m.applied))
	case m.errText != "":
		return m.st.warn.Render("! " + m.errText)
	case m.busy && m.snap == nil:
		return m.st.dimText.Render(m.spin.View() + " lighting the lanterns…")
	default:
		return strings.Join([]string{
			lipgloss.NewStyle().Foreground(lantern).Render("◉"),
			"",
			m.st.empty.Render("No spirits lingering. The river is clear."),
			m.st.dimText.Render("r rescan · q quit"),
		}, "\n")
	}
}

func (m Model) contextLine() string {
	if m.toast != "" {
		return m.st.toast.Render(truncCell(m.toast, m.contentW()))
	}
	var parts []string
	if m.filtering {
		parts = append(parts, m.st.filterChip.Render(" / ")+" "+m.filter.View())
	} else if m.applied != "" {
		parts = append(parts, m.st.filterChip.Render(" /"+m.applied+" "))
	}
	info := m.st.dimText.Render(fmt.Sprintf("%d shown", len(m.view)))
	if selN := len(m.select_); selN > 0 {
		info += m.st.faintText.Render(" · ") +
			m.st.selDot.Render(fmt.Sprintf("%d selected", selN))
	}
	if len(m.view) > 0 && m.cursor < len(m.view) {
		if cmd := m.view[m.cursor].PIDs[0].Process.Cmdline; cmd != "" {
			budget := m.contentW() - lipgloss.Width(info) - 4
			for _, p := range parts {
				budget -= lipgloss.Width(p) + 2
			}
			if budget > 10 {
				info += "  " + m.st.faintText.Render("$ "+truncCell(cmd, budget-2))
			}
		}
	}
	parts = append(parts, info)
	return strings.Join(parts, "  ")
}

// hk is one key hint in the footer.
type hk struct{ k, v string }

func (m Model) hintLine() string {
	var keys []hk
	switch {
	case m.confirm != nil:
		keys = []hk{{"y", "send off"}, {"n/esc", "keep them"}}
	case m.filtering:
		keys = []hk{{"enter", "apply"}, {"esc", "cancel"}}
	case m.detail != nil:
		keys = []hk{{"o", "open in browser"}, {"j/k", "scroll"}, {"esc", "close"}}
		if m.detail.Proto != scan.TCP {
			keys = keys[1:]
		}
	default:
		// widest set that fits this terminal
		sets := [][]hk{
			{
				{"j/k", "move"}, {"space", "select"}, {"x", "send off"},
				{"enter", "detail"}, {"o", "open"}, {"/", "filter"},
				{"s", "sort"}, {"r", "rescan"}, {"?", "help"}, {"q", "quit"},
			},
			{
				{"j/k", "move"}, {"space", "select"}, {"x", "send off"},
				{"/", "filter"}, {"?", "help"}, {"q", "quit"},
			},
			{{"x", "send off"}, {"?", "help"}, {"q", "quit"}},
			{{"?", "help"}},
		}
		for _, set := range sets {
			if hintWidth(set) <= m.contentW() {
				keys = set
				break
			}
		}
	}
	var out strings.Builder
	for i, k := range keys {
		if i > 0 {
			out.WriteString(m.st.faintText.Render(" · "))
		}
		out.WriteString(m.st.helpKey.Render(k.k))
		out.WriteString(m.st.help.Render(" " + k.v))
	}
	return out.String()
}

func hintWidth(keys []hk) int {
	w := 0
	for i, k := range keys {
		if i > 0 {
			w += 3 // " · "
		}
		w += len(k.k) + 1 + len(k.v)
	}
	return w
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

func padLeft(s string, w int) string {
	l := lipgloss.Width(s)
	if l >= w {
		return s
	}
	return strings.Repeat(" ", w-l) + s
}

// truncCell shortens plain cell text, keeping the head.
func truncCell(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 1 {
		return string(r[:maxInt(w, 0)])
	}
	return string(r[:w-1]) + "…"
}

// truncRunes keeps the old name for the head-truncating helper.
func truncRunes(s string, w int) string { return truncCell(s, w) }

// truncPath shortens a path from the left, keeping the tail: the
// project name matters more than the /Users prefix.
func truncPath(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 1 {
		return string(r[:maxInt(w, 0)])
	}
	return "…" + string(r[len(r)-w+1:])
}

// tildeHome abbreviates the current home directory to ~ for display.
func tildeHome(p, home string) string {
	if home == "" || !strings.HasPrefix(p, home) {
		return p
	}
	rest := strings.TrimPrefix(p, home)
	if rest == "" {
		return "~"
	}
	if !strings.HasPrefix(rest, "/") {
		return p
	}
	return "~" + rest
}

func userHome() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}
