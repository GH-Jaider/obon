package ui

import (
	"fmt"
	"strings"

	"github.com/GH-Jaider/obon/internal/scan"
)

// renderConfirm builds the send-off dialog: the one place the metaphor
// speaks plainly: a send off, not a kill. Each target carries its
// safety verdict; the dialog warns before anything load-bearing goes.
func (m Model) renderConfirm() string {
	ts := m.confirm.targets
	w := minInt(m.width-8, 60)
	if w < 40 {
		w = 40
	}
	inner := w - 6 // text budget inside the box padding

	var b strings.Builder
	b.WriteString(m.st.modalTitle.Render(" " + sendOffTitle(len(ts)) + " "))
	b.WriteString("\n\n")
	const maxShown = 8
	for i, t := range ts {
		if i == maxShown && len(ts) > maxShown {
			rest := len(ts) - maxShown
			b.WriteString(m.st.dimText.Render(fmt.Sprintf("  … and %d more", rest)))
			b.WriteString("\n")
			break
		}
		pill := m.safetyPill(scan.SafetyUnknown)
		if i < len(m.confirm.safety) {
			pill = m.safetyPill(m.confirm.safety[i].Level)
		}
		portPart := ""
		if t.Port > 0 {
			portPart = " :" + fmt.Sprint(t.Port)
		}
		b.WriteString(pill)
		b.WriteString(" ")
		b.WriteString(m.st.row.Render(truncCell(
			fmt.Sprintf("%s%s", t.Name, portPart), inner-16)))
		b.WriteString(m.st.dimText.Render(fmt.Sprintf("  pid %d", t.PID)))
		b.WriteString("\n")
	}

	if warn := m.confirmWarning(inner); warn != "" {
		b.WriteString("\n")
		b.WriteString(warn)
	}
	b.WriteString("\n")
	b.WriteString(m.st.dimText.Render("SIGTERM first · SIGKILL if they linger"))
	return m.st.modalBox.Width(w).Render(b.String())
}

// confirmWarning sums up the verdicts: one line that says what the
// send-off will actually cost, worst news first.
func (m Model) confirmWarning(w int) string {
	var sys, app, unk int
	for _, s := range m.confirm.safety {
		switch s.Level {
		case scan.SafetySystem:
			sys++
		case scan.SafetyCaution:
			app++
		case scan.SafetyUnknown:
			unk++
		}
	}
	switch {
	case sys > 0:
		msg := fmt.Sprintf("%d system %s here — macOS features may break; launchd usually relights them",
			sys, pluralNoun(sys, "service", "services"))
		return m.st.safeSys.Render(wrap(msg, w)) + "\n"
	case app > 0:
		msg := fmt.Sprintf("%d app %s here — the parent app may misbehave or crash",
			app, pluralNoun(app, "helper", "helpers"))
		return m.st.safeWarn.Render(wrap(msg, w)) + "\n"
	case unk > 0:
		msg := fmt.Sprintf("%d unrecognised — check the detail panel first if unsure",
			unk)
		return m.st.dimText.Render(wrap(msg, w)) + "\n"
	case len(m.confirm.safety) > 0:
		return m.st.safeOK.Render("all dev servers — nothing else depends on them") + "\n"
	}
	return ""
}

func pluralNoun(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func sendOffTitle(n int) string {
	if n == 1 {
		return "Send off this process?"
	}
	return fmt.Sprintf("Send off %d processes?", n)
}

// renderHelp lists every key, quietly, plus the safety legend.
func (m Model) renderHelp() string {
	rows := [][2]string{
		{"j / k, arrows", "move cursor"},
		{"g / G", "top / bottom"},
		{"pgup / pgdn", "page"},
		{"enter", "detail panel"},
		{"o", "open localhost:port in browser"},
		{"space", "toggle selection"},
		{"a / A", "select all visible / clear"},
		{"x", "send off selection or row"},
		{"/", "filter the board"},
		{"esc", "clear selection, then filter"},
		{"s / S", "cycle sort / flip direction"},
		{"r", "rescan now"},
		{"?", "toggle this help"},
		{"q", "quit obon"},
	}
	var b strings.Builder
	b.WriteString(m.st.modalTitle.Render(" keys "))
	b.WriteString("\n\n")
	for _, r := range rows {
		b.WriteString(m.st.helpKey.Render(padRight(r[0], 16)))
		b.WriteString(m.st.help.Render(r[1]))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	legend := []struct {
		level scan.SafetyLevel
		text  string
	}{
		{scan.SafetySafe, "safe to send off"},
		{scan.SafetyCaution, "an app leans on it"},
		{scan.SafetySystem, "the OS leans on it"},
		{scan.SafetyUnknown, "unrecognised"},
	}
	for _, l := range legend {
		b.WriteString(m.safetyPill(l.level))
		b.WriteString("  ")
		b.WriteString(m.st.help.Render(l.text))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.st.dimText.Render("* after proto = bound to all interfaces"))
	return m.st.modalBox.Width(52).Render(b.String())
}
