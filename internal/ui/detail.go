package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/GH-Jaider/obon/internal/scan"
)

// showURL: a browsable link makes sense for TCP dev servers.
func (m Model) showURL(g *scan.Group) bool {
	return g.Proto == scan.TCP && g.PIDs[0].Safety.Level == scan.SafetySafe
}

// renderDetail builds the detail modal for the cursor row: the URL and
// what it serves, the send-off verdict, full command, cwd, start time,
// parent chain as a tree, and every socket of the PID.
func (m Model) renderDetail() string {
	g := m.detail
	if g == nil {
		return ""
	}
	w := minInt(m.width-8, 84)
	if w < 40 {
		w = 40
	}
	cw := w - 6 // text budget inside the box padding

	var b strings.Builder
	title := g.PIDs[0].Process.Name + " :" + fmt.Sprint(g.Port)
	b.WriteString(m.st.modalTitle.Render(" " + title + " "))
	b.WriteString("\n")
	if m.showURL(g) {
		b.WriteString(m.urlBlock(g, cw))
	}

	for _, d := range g.PIDs {
		b.WriteString("\n")
		b.WriteString(m.safetyBlock(d, cw))
		b.WriteString("\n")
		b.WriteString(section("process"))
		b.WriteString(kv("pid", fmt.Sprint(d.Socket.PID), cw))
		if d.Process.User != "" {
			b.WriteString(kv("user", d.Process.User, cw))
		}
		b.WriteString(kv("origin", d.Origin, cw))
		if !d.Process.StartedAt.IsZero() {
			st := d.Process.StartedAt.Format("15:04:05 · Jan 2")
			b.WriteString(kv("started", st+"  ("+humanDur(d.Uptime)+" ago)", cw))
		}
		if cmd := d.Process.Cmdline; cmd != "" {
			b.WriteString(labeled("command", wrap(cmd, cw-10), cw))
		} else {
			b.WriteString(kv("command", "(unavailable)", cw))
		}
		if cwd := d.Process.Cwd; cwd != "" {
			b.WriteString(labeled("cwd", wrap(tildeHome(cwd, m.home), cw-10), cw))
		}
		if socks := socketsOf(m.snap, d.Socket.PID); len(socks) > 0 {
			var ss []string
			for _, s := range socks {
				ss = append(ss, fmt.Sprintf("%s :%d @ %s", strings.ToUpper(s.Proto), s.Port, s.Bind))
			}
			b.WriteString(labeled("sockets", strings.Join(ss, "\n"), cw))
		}
		b.WriteString(m.chainBlock(d, cw))
	}

	return m.st.modalBox.Width(w).Render(b.String())
}

// urlBlock: the clickable localhost link and what answered there.
func (m Model) urlBlock(g *scan.Group, cw int) string {
	url := localURL(g.Port)
	link := hyperlink(url, lipgloss.NewStyle().Foreground(violet).Underline(true).Render(url))
	var b strings.Builder
	b.WriteString(link)
	b.WriteString(m.st.faintText.Render("  · o opens it"))
	b.WriteString("\n")
	b.WriteString(m.servingLine(g, cw))
	b.WriteString("\n")
	return b.String()
}

// servingLine reports the probe: status, content type and page title.
func (m Model) servingLine(g *scan.Group, cw int) string {
	if m.probeKey != g.Key || m.probe == nil {
		return m.st.faintText.Render("knocking on the door…")
	}
	p := m.probe
	if p.err != nil {
		return m.st.faintText.Render("no http reply — not a web server, or it speaks another protocol")
	}
	sty := m.st.safeOK
	switch {
	case p.status >= 500:
		sty = m.st.safeSys
	case p.status >= 300:
		sty = m.st.safeWarn
	}
	out := sty.Render(fmt.Sprint(p.status))
	if p.ctype != "" {
		out += m.st.dimText.Render(" · " + p.ctype)
	}
	if p.title != "" {
		budget := cw - lipgloss.Width(out) - 3
		if budget > 8 {
			out += m.st.dimText.Render(" · ") + m.st.row.Render("«"+truncCell(p.title, budget-2)+"»")
		}
	}
	return out
}

// safetyBlock answers the question the board exists for: what happens
// if you send this one off.
func (m Model) safetyBlock(d *scan.Detail, cw int) string {
	s := d.Safety
	var b strings.Builder
	b.WriteString(section("send-off"))
	b.WriteString(padRight(dimStyle().Render("verdict")+": ", 10))
	b.WriteString(m.safetyPill(s.Level))
	b.WriteString("  ")
	b.WriteString(m.safetyStyle(s.Level).Render(truncCell(s.Reason, maxInt(8, cw-17))))
	b.WriteString("\n")
	b.WriteString(labeled("if sent", wrap(s.Consequence, cw-10), cw))
	return b.String()
}

// chainBlock renders the parent tree, root first, listener last.
func (m Model) chainBlock(d *scan.Detail, cw int) string {
	if len(d.Parents) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(section("parent chain"))

	chain := make([]scan.ProcRef, len(d.Parents))
	copy(chain, d.Parents)
	// reverse: root first
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	for i, ref := range chain {
		indent := strings.Repeat("   ", i)
		conn := ""
		if i > 0 {
			conn = "└─ "
		}
		mark := ""
		if i == len(chain)-1 && d.Origin != "manual" {
			mark = m.st.originAg.Render(" ← lit this one")
		}
		line := fmt.Sprintf("%s%s%s (%d)%s", indent, conn, ref.Name, ref.PID, mark)
		b.WriteString(truncRunes(line, cw))
		b.WriteString("\n")
	}
	listener := fmt.Sprintf("%s└─ %s (%d)", strings.Repeat("   ", len(chain)),
		d.Process.Name, d.Socket.PID)
	b.WriteString(m.st.row.Render(truncRunes(listener, cw)))
	b.WriteString("\n")
	return b.String()
}

func section(name string) string {
	return dimStyle().Render(strings.ToUpper(name)) + "\n"
}

func kv(k, v string, cw int) string {
	return padRight(dimStyle().Render(k)+": ", 10) + truncRunes(v, maxInt(8, cw-10)) + "\n"
}

func labeled(label, body string, cw int) string {
	var out strings.Builder
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		if i == 0 {
			out.WriteString(padRight(dimStyle().Render(label)+": ", 10))
		} else {
			out.WriteString(strings.Repeat(" ", 10))
		}
		out.WriteString(truncRunes(l, maxInt(8, cw-10)))
		out.WriteString("\n")
	}
	return out.String()
}

func wrap(s string, width int) string {
	if width < 10 {
		width = 10
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := ""
		for _, wd := range words {
			// hard-break words longer than a full line (paths, hashes)
			for len(wd) > width {
				if line != "" {
					out = append(out, line)
					line = ""
				}
				out = append(out, wd[:width])
				wd = wd[width:]
			}
			switch {
			case line == "":
				line = wd
			case len(line)+1+len(wd) <= width:
				line += " " + wd
			default:
				out = append(out, line)
				line = wd
			}
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// socketsOf collects every socket held by pid across the snapshot.
func socketsOf(snap *scan.Snapshot, pid int32) []scan.Socket {
	var out []scan.Socket
	if snap == nil {
		return out
	}
	for _, gr := range snap.Groups {
		for _, dd := range gr.PIDs {
			if dd.Socket.PID == pid {
				out = append(out, dd.Socket)
			}
		}
	}
	return out
}
