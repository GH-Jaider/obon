package ui

import (
	"fmt"
	"strings"

	"github.com/GH-Jaider/obon/internal/scan"
)

// renderDetail builds the detail modal for the cursor row: full command,
// cwd, start time, parent chain as a tree, and every socket of the PID.
func (m Model) renderDetail() string {
	g := m.detail
	if g == nil {
		return ""
	}
	w := minInt(m.width-8, 84)
	if w < 40 {
		w = 40
	}

	var b strings.Builder
	title := g.PIDs[0].Process.Name + " :" + fmt.Sprint(g.Port)
	b.WriteString(m.st.modalTitle.Render(title))
	b.WriteString("\n")

	for _, d := range g.PIDs {
		b.WriteString("\n")
		b.WriteString(section("process", w))
		b.WriteString(kv("pid", fmt.Sprint(d.Socket.PID), w))
		if d.Process.User != "" {
			b.WriteString(kv("user", d.Process.User, w))
		}
		b.WriteString(kv("origin", d.Origin, w))
		if !d.Process.StartedAt.IsZero() {
			st := d.Process.StartedAt.Format("15:04:05 · Jan 2")
			b.WriteString(kv("started", st+"  ("+humanDur(d.Uptime)+" ago)", w))
		}
		if cmd := d.Process.Cmdline; cmd != "" {
			b.WriteString(labeled("command", wrap(cmd, w-12), w))
		} else {
			b.WriteString(kv("command", "(unavailable)", w))
		}
		if cwd := d.Process.Cwd; cwd != "" {
			b.WriteString(labeled("cwd", wrap(cwd, w-12), w))
		}
		if socks := socketsOf(m.snap, d.Socket.PID); len(socks) > 0 {
			if len(socks) > 0 {
				var ss []string
				for _, s := range socks {
					ss = append(ss, fmt.Sprintf("%s :%d @ %s", strings.ToUpper(s.Proto), s.Port, s.Bind))
				}
				b.WriteString(labeled("sockets", strings.Join(ss, "\n"), w))
			}
		}
		b.WriteString(m.chainBlock(d, w))
	}

	return m.st.modalBox.Width(w).Render(b.String())
}

// chainBlock renders the parent tree, root first, listener last.
func (m Model) chainBlock(d *scan.Detail, w int) string {
	if len(d.Parents) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(section("parent chain", w))

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
		b.WriteString(truncRunes(line, w-4))
		b.WriteString("\n")
	}
	listener := fmt.Sprintf("%s└─ %s (%d)", strings.Repeat("   ", len(chain)),
		d.Process.Name, d.Socket.PID)
	b.WriteString(m.st.row.Render(truncRunes(listener, w-4)))
	b.WriteString("\n")
	return b.String()
}

func section(name string, _ int) string {
	return dimStyle().Render(strings.ToUpper(name)) + "\n"
}

func kv(k, v string, w int) string {
	return padRight(dimStyle().Render(k)+": ", 10) + truncRunes(v, maxInt(8, w-14)) + "\n"
}

func labeled(label, body string, w int) string {
	var out strings.Builder
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		if i == 0 {
			out.WriteString(padRight(dimStyle().Render(label)+": ", 10))
		} else {
			out.WriteString(strings.Repeat(" ", 10))
		}
		out.WriteString(truncRunes(l, maxInt(8, w-14)))
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
		line := words[0]
		for _, wd := range words[1:] {
			if len(line)+1+len(wd) <= width {
				line += " " + wd
			} else {
				out = append(out, line)
				line = wd
			}
		}
		out = append(out, line)
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
