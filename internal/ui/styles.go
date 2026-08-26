// Package ui is the obon TUI: the lantern board.
//
// Visual identity: one warm accent — a paper-lantern glow — over cool,
// dim night-river neutrals. The lantern is the only warm thing on
// screen; it marks what was just lit and who summoned it. Everything
// else stays quiet. No emoji, no ASCII art in data cells, no themed
// column names: Port · Proto · Process · PID · Origin · Uptime · CWD.
package ui

import "github.com/charmbracelet/lipgloss"

// Adaptive lantern palette. Dark terminals get a soft warm glow;
// light ones get a deep amber that reads as ink, not neon.
var (
	accent    = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FFB86C"} // lantern
	accentDim = lipgloss.AdaptiveColor{Light: "#92610F", Dark: "#8A6D3B"} // lantern at dusk

	base = lipgloss.AdaptiveColor{Light: "#33363E", Dark: "#C6CCDA"}
	dim     = lipgloss.AdaptiveColor{Light: "#80838B", Dark: "#5D6572"}
	faint   = lipgloss.AdaptiveColor{Light: "#C4C1BA", Dark: "#39404E"}

	// flashInk sits on top of the accent when a row is freshly lit.
	flashInk = lipgloss.AdaptiveColor{Light: "#FFF8EF", Dark: "#201507"}
)

// styles groups every rendered surface so the look lives in one place.
type styles struct {
	title      lipgloss.Style
	subtitle   lipgloss.Style
	header     lipgloss.Style
	headerSort lipgloss.Style // active sort column title
	row        lipgloss.Style
	rowSel     lipgloss.Style // cursor row
	rowAgent   lipgloss.Style // spawned by an agent: slightly brighter, indigo-tinged
	rowFlash   lipgloss.Style // just lit: warm highlight
	rowGone    lipgloss.Style // departing: drifting away
	originMan  lipgloss.Style // "manual" origin
	originAg   lipgloss.Style // agent origin: the lantern's owner
	warn       lipgloss.Style // 0.0.0.0 marker
	dimText    lipgloss.Style
	toast      lipgloss.Style
	modalBox   lipgloss.Style // rounded, accent-dim border
	modalTitle lipgloss.Style
	filterBox  lipgloss.Style
	help       lipgloss.Style
	helpKey    lipgloss.Style
	empty      lipgloss.Style
}

func newStyles() styles {
	return styles{
		title:      lipgloss.NewStyle().Bold(true).Foreground(accent),
		subtitle:   lipgloss.NewStyle().Foreground(dim),
		header:     lipgloss.NewStyle().Foreground(dim).Bold(true),
		headerSort: lipgloss.NewStyle().Foreground(accentDim).Bold(true),
		row:        lipgloss.NewStyle().Foreground(base),
		rowSel:     lipgloss.NewStyle().Foreground(base).Background(faint),
		rowAgent:   lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#1F2430", Dark: "#E8ECF5"}),
		rowFlash:   lipgloss.NewStyle().Foreground(flashInk).Background(accent).Bold(true),
		rowGone:    lipgloss.NewStyle().Foreground(faint).Italic(true),
		originMan:  lipgloss.NewStyle().Foreground(dim),
		originAg:   lipgloss.NewStyle().Foreground(accentDim),
		warn:       lipgloss.NewStyle().Foreground(dim).Bold(true),
		dimText:    lipgloss.NewStyle().Foreground(dim),
		toast:      lipgloss.NewStyle().Foreground(accent).Bold(true),
		modalBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentDim).
			Padding(1, 3).
			MarginTop(1),
		modalTitle: lipgloss.NewStyle().Bold(true).Foreground(accent),
		filterBox:  lipgloss.NewStyle().Foreground(accent),
		help:       lipgloss.NewStyle().Foreground(dim),
		helpKey:    lipgloss.NewStyle().Foreground(accentDim).Bold(true),
		empty:      lipgloss.NewStyle().Foreground(dim).Italic(true),
	}
}

func dimStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(dim) }
