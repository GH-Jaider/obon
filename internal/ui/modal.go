package ui

import (
	"fmt"
	"strings"
)

// renderConfirm builds the send-off dialog: the one place the metaphor
// speaks plainly — a send-off, not a kill.
func (m Model) renderConfirm() string {
	ts := m.confirm.targets
	w := minInt(m.width-8, 56)
	if w < 40 {
		w = 40
	}
	var b strings.Builder
	b.WriteString(m.st.modalTitle.Render(sendOffTitle(len(ts))))
	b.WriteString("\n\n")
	const maxShown = 9
	for i, t := range ts {
		if i == maxShown && len(ts) > maxShown {
			rest := len(ts) - maxShown
			b.WriteString(m.st.dimText.Render(fmt.Sprintf("… and %d more", rest)))
			b.WriteString("\n")
			break
		}
		portPart := ""
		if t.Port > 0 {
			portPart = " :" + fmt.Sprint(t.Port)
		}
		b.WriteString(m.st.row.Render(truncRunes(
			fmt.Sprintf("%s%s  (pid %d)", t.Name, portPart, t.PID), w)))
		b.WriteString("\n")
	}
	if len(ts) <= maxShown {
		b.WriteString("\n")
	}
	b.WriteString(m.st.dimText.Render("SIGTERM first · SIGKILL if they linger"))
	return m.st.modalBox.Width(w).Render(b.String())
}

func sendOffTitle(n int) string {
	if n == 1 {
		return "Send off this process?"
	}
	return fmt.Sprintf("Send off %d processes?", n)
}

// renderHelp lists every key, quietly.
func (m Model) renderHelp() string {
	rows := [][2]string{
		{"j / k, arrows", "move cursor"},
		{"g / G", "top / bottom"},
		{"pgup / pgdn", "page"},
		{"enter", "detail panel"},
		{"space", "toggle selection"},
		{"a / A", "select all visible / clear"},
		{"x", "send off selection or row"},
		{"/", "filter port·process·cmdline·cwd·origin"},
		{"esc", "clear selection, then filter"},
		{"s / S", "cycle sort column / flip direction"},
		{"r", "rescan now"},
		{"?", "toggle this help"},
		{"q", "quit obon"},
	}
	var b strings.Builder
	b.WriteString(m.st.modalTitle.Render("keys"))
	b.WriteString("\n\n")
	for _, r := range rows {
		b.WriteString(m.st.helpKey.Render(padRight(r[0], 16)))
		b.WriteString(m.st.help.Render(r[1]))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.st.dimText.Render("* after PROTO = bound to all interfaces"))
	return m.st.modalBox.Width(50).Render(b.String())
}
