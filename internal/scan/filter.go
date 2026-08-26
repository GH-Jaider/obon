package scan

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Filter matches groups against a raw query: regex when it compiles,
// case-insensitive substring otherwise.
type Filter struct {
	Raw string
	re  *regexp.Regexp
}

// CompileFilter builds a Filter from raw text. Empty text matches everything.
func CompileFilter(raw string) *Filter {
	f := &Filter{Raw: strings.TrimSpace(raw)}
	if f.Raw == "" {
		return f
	}
	if re, err := regexp.Compile("(?i)" + f.Raw); err == nil {
		f.re = re
	}
	return f
}

// Match reports whether the group's port, process names, command lines,
// cwd, user or origin fields satisfy the filter.
func (f *Filter) Match(g *Group) bool {
	if f.Raw == "" {
		return true
	}
	if f.matchText(strconv.Itoa(g.Port)) || f.matchText(g.Proto) {
		return true
	}
	for _, b := range g.Binds {
		if f.matchText(b) {
			return true
		}
	}
	for _, d := range g.PIDs {
		if f.matchText(d.Process.Name) ||
			f.matchText(d.Process.Cmdline) ||
			f.matchText(d.Process.Cwd) ||
			f.matchText(d.Process.User) ||
			f.matchText(d.Origin) ||
			f.matchText(d.Safety.Label) {
			return true
		}
	}
	return false
}

func (f *Filter) matchText(s string) bool {
	if s == "" {
		return false
	}
	if f.re != nil {
		return f.re.MatchString(s)
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(f.Raw))
}

// SortKey identifies sortable columns.
type SortKey int

// Sort keys.
const (
	SortPort SortKey = iota
	SortProto
	SortProcess
	SortPID
	SortOrigin
	SortUptime
	SortSafety
)

// ParseSortKey maps a column name to its key; ok=false if unknown.
func ParseSortKey(name string) (SortKey, bool) {
	switch strings.ToLower(name) {
	case "port":
		return SortPort, true
	case "proto":
		return SortProto, true
	case "process", "name":
		return SortProcess, true
	case "pid":
		return SortPID, true
	case "origin":
		return SortOrigin, true
	case "uptime", "age":
		return SortUptime, true
	case "safety", "safe":
		return SortSafety, true
	}
	return SortPort, false
}

// SortGroups sorts in place; ties broken by port then proto for stable rows.
func SortGroups(gs []*Group, key SortKey, asc bool) {
	sort.Slice(gs, func(i, j int) bool {
		c := compareGroups(gs[i], gs[j], key)
		if c == 0 {
			c = compareGroups(gs[i], gs[j], SortPort)
		}
		if c == 0 {
			c = strings.Compare(gs[i].Proto, gs[j].Proto)
		}
		if !asc {
			c = -c
		}
		return c < 0
	})
}

func compareGroups(a, b *Group, key SortKey) int {
	x, y := a.primary(), b.primary()
	switch key {
	case SortPort:
		return cmpInt(a.Port, b.Port)
	case SortProto:
		return strings.Compare(a.Proto, b.Proto)
	case SortProcess:
		return strings.Compare(strings.ToLower(x.Process.Name), strings.ToLower(y.Process.Name))
	case SortPID:
		return cmpInt(int(x.Socket.PID), int(y.Socket.PID))
	case SortOrigin:
		return strings.Compare(x.Origin, y.Origin)
	case SortUptime:
		return cmpDur(x.Uptime, y.Uptime)
	case SortSafety:
		return cmpInt(x.Safety.Level.Rank(), y.Safety.Level.Rank())
	}
	return 0
}

func (g *Group) primary() *Detail {
	if len(g.PIDs) > 0 {
		return g.PIDs[0]
	}
	return &Detail{}
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func cmpDur(a, b time.Duration) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}
