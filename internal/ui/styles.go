// Package ui is the obon TUI: the lantern board.
//
// Visual identity: charm-school confidence with the paper lantern as
// the brand. The lantern orange badges the title, the cursor bar and
// freshly-lit rows; violet marks agents and the cursor's water; safety
// verdicts are solid-color pills: mint, sun, ember. The whole board
// lives inside a rounded frame. No emoji, no themed column names.
package ui

import "github.com/charmbracelet/lipgloss"

// Adaptive palette. Dark terminals get the night river; light ones the
// same inks deepened so they read on paper.
var (
	// the lantern: obon's brand orange
	lantern    = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FFB86C"}
	lanternInk = lipgloss.AdaptiveColor{Light: "#FFF8EF", Dark: "#2A1804"} // text on lantern

	// charm violet: agents, the cursor's water, modal frames
	violet   = lipgloss.AdaptiveColor{Light: "#6D28D9", Dark: "#A78BFA"}
	cursorBg = lipgloss.AdaptiveColor{Light: "#EDE7FA", Dark: "#312D5A"}

	// pink: selection
	pink = lipgloss.AdaptiveColor{Light: "#BE185D", Dark: "#F780D8"}

	// neutrals, brighter than a whisper
	text    = lipgloss.AdaptiveColor{Light: "#1F2430", Dark: "#E8ECF5"}
	subtext = lipgloss.AdaptiveColor{Light: "#4B5263", Dark: "#A9B1C3"}
	muted   = lipgloss.AdaptiveColor{Light: "#7A8090", Dark: "#7A8399"}
	faint   = lipgloss.AdaptiveColor{Light: "#C9CCD4", Dark: "#454D5F"}

	// flashInk sits on top of the lantern when a row is freshly lit.
	flashInk = lipgloss.AdaptiveColor{Light: "#FFF8EF", Dark: "#2A1804"}

	// safety pill inks: mint (send off freely), sun (an app leans on
	// it), ember (the OS leans on it), slate (unrecognised).
	mintBg  = lipgloss.AdaptiveColor{Light: "#2F7D4F", Dark: "#84D996"}
	mintInk = lipgloss.AdaptiveColor{Light: "#F2FBF5", Dark: "#0D2818"}
	sunBg   = lipgloss.AdaptiveColor{Light: "#B07C1E", Dark: "#F2C879"}
	sunInk  = lipgloss.AdaptiveColor{Light: "#FFFDF6", Dark: "#2E1F04"}
	emberBg = lipgloss.AdaptiveColor{Light: "#C24532", Dark: "#FF8B7A"}
	emberIn = lipgloss.AdaptiveColor{Light: "#FFF6F4", Dark: "#330F08"}
	slateBg = lipgloss.AdaptiveColor{Light: "#8B92A3", Dark: "#4C5468"}
	slateIn = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#D6DCE8"}

	// plain foregrounds for prose (modal reasons, warnings)
	mint  = lipgloss.AdaptiveColor{Light: "#2F7D4F", Dark: "#84D996"}
	sun   = lipgloss.AdaptiveColor{Light: "#B07C1E", Dark: "#F2C879"}
	ember = lipgloss.AdaptiveColor{Light: "#C24532", Dark: "#FF8B7A"}
)

// styles groups every rendered surface so the look lives in one place.
type styles struct {
	titleBadge lipgloss.Style // " ◉ obon " on lantern orange
	subtitle   lipgloss.Style
	border     lipgloss.Style // the rounded frame around the board
	header     lipgloss.Style
	headerSort lipgloss.Style // active sort column title
	rule       lipgloss.Style // faint horizontal separators
	row        lipgloss.Style
	rowAgent   lipgloss.Style // spawned by an agent: violet-tinged
	rowFlash   lipgloss.Style // just lit: the lantern passes over it
	rowGone    lipgloss.Style // departing: drifting away
	originAg   lipgloss.Style // agent origin: the lantern's owner
	warn       lipgloss.Style // 0.0.0.0 marker
	dimText    lipgloss.Style
	faintText  lipgloss.Style
	selDot     lipgloss.Style // pink selection dot
	toast      lipgloss.Style
	modalBox   lipgloss.Style // rounded violet frame
	modalTitle lipgloss.Style // badge on lantern orange
	filterChip lipgloss.Style // active filter, violet pill
	help       lipgloss.Style
	helpKey    lipgloss.Style
	empty      lipgloss.Style

	pillDev lipgloss.Style // mint pill: safe to send off
	pillApp lipgloss.Style // sun pill: an app depends on it
	pillSys lipgloss.Style // ember pill: the OS depends on it
	pillUnk lipgloss.Style // slate pill: unrecognised

	safeOK   lipgloss.Style // prose in mint
	safeWarn lipgloss.Style // prose in sun
	safeSys  lipgloss.Style // prose in ember
	safeUnk  lipgloss.Style // prose in muted
}

func newStyles() styles {
	return styles{
		titleBadge: lipgloss.NewStyle().Bold(true).Foreground(lanternInk).Background(lantern),
		subtitle:   lipgloss.NewStyle().Foreground(muted),
		border:     lipgloss.NewStyle().Foreground(faint),
		header:     lipgloss.NewStyle().Foreground(muted).Bold(true),
		headerSort: lipgloss.NewStyle().Foreground(lantern).Bold(true),
		rule:       lipgloss.NewStyle().Foreground(faint),
		row:        lipgloss.NewStyle().Foreground(text),
		rowAgent:   lipgloss.NewStyle().Foreground(violet),
		rowFlash:   lipgloss.NewStyle().Foreground(flashInk).Background(lantern).Bold(true),
		rowGone:    lipgloss.NewStyle().Foreground(faint).Italic(true),
		originAg:   lipgloss.NewStyle().Foreground(violet).Bold(true),
		warn:       lipgloss.NewStyle().Foreground(sun).Bold(true),
		dimText:    lipgloss.NewStyle().Foreground(subtext),
		faintText:  lipgloss.NewStyle().Foreground(muted),
		selDot:     lipgloss.NewStyle().Foreground(pink).Bold(true),
		toast:      lipgloss.NewStyle().Foreground(lantern).Bold(true),
		modalBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(violet).
			Padding(1, 3),
		modalTitle: lipgloss.NewStyle().Bold(true).Foreground(lanternInk).Background(lantern),
		filterChip: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1B1730"}).Background(violet),
		help:       lipgloss.NewStyle().Foreground(muted),
		helpKey:    lipgloss.NewStyle().Foreground(lantern).Bold(true),
		empty:      lipgloss.NewStyle().Foreground(muted).Italic(true),

		pillDev: lipgloss.NewStyle().Bold(true).Foreground(mintInk).Background(mintBg),
		pillApp: lipgloss.NewStyle().Bold(true).Foreground(sunInk).Background(sunBg),
		pillSys: lipgloss.NewStyle().Bold(true).Foreground(emberIn).Background(emberBg),
		pillUnk: lipgloss.NewStyle().Bold(true).Foreground(slateIn).Background(slateBg),

		safeOK:   lipgloss.NewStyle().Foreground(mint),
		safeWarn: lipgloss.NewStyle().Foreground(sun),
		safeSys:  lipgloss.NewStyle().Foreground(ember),
		safeUnk:  lipgloss.NewStyle().Foreground(muted),
	}
}

func dimStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(subtext) }
