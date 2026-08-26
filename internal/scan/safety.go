package scan

import (
	"os"
	"os/user"
	"strings"
)

// SafetyLevel ranks how risky sending off a listener is.
type SafetyLevel int

// Safety levels, from harmless to load-bearing.
const (
	SafetyUnknown SafetyLevel = iota // unrecognised: look before you leap
	SafetySafe                       // dev server / agent-lit: nothing else depends on it
	SafetyCaution                    // piece of a desktop app: the app may break
	SafetySystem                     // OS daemon: features break, launchd relights it
)

// String returns the JSON/CLI name of the level.
func (l SafetyLevel) String() string {
	switch l {
	case SafetySafe:
		return "safe"
	case SafetyCaution:
		return "caution"
	case SafetySystem:
		return "system"
	}
	return "unknown"
}

// Rank orders levels safest-first for sorting.
func (l SafetyLevel) Rank() int {
	switch l {
	case SafetySafe:
		return 0
	case SafetyUnknown:
		return 1
	case SafetyCaution:
		return 2
	case SafetySystem:
		return 3
	}
	return 1
}

// Safety explains what sending off a process would actually do.
type Safety struct {
	Level       SafetyLevel `json:"-"`
	Label       string      `json:"level"`       // "safe" | "caution" | "system" | "unknown"
	Reason      string      `json:"reason"`      // why it was classified this way
	Consequence string      `json:"consequence"` // what happens if you send it off
}

// SafetyEnv is the caller's context for classification; injectable in tests.
type SafetyEnv struct {
	Home string // current user's home directory ("" = unknown)
	User string // current username ("" = unknown)
}

// CurrentSafetyEnv reads the real environment once per scanner.
func CurrentSafetyEnv() SafetyEnv {
	env := SafetyEnv{}
	if h, err := os.UserHomeDir(); err == nil {
		env.Home = h
	}
	if u, err := user.Current(); err == nil {
		env.User = u.Username
	}
	return env
}

// systemDaemons maps well-known macOS daemon names to what breaks
// when they go. All of these are relit by launchd on demand.
var systemDaemons = map[string]string{
	"rapportd":          "Continuity: Handoff, Universal Control and AirDrop",
	"sharingd":          "AirDrop and local sharing",
	"ControlCenter":     "the menu-bar Control Center",
	"replicatord":       "Sidecar and device replication",
	"remoted":           "Apple remote-device services",
	"mDNSResponder":     "Bonjour and .local DNS for the whole machine",
	"airportd":          "Wi-Fi management",
	"bluetoothd":        "Bluetooth",
	"identityservicesd": "iMessage and FaceTime identity",
	"AirPlayXPCHelper":  "AirPlay",
	"WindowServer":      "every window on screen",
	"launchd":           "the entire user session",
}

// devRuntimes are interpreter/runtime names that almost always mean
// "a dev server someone started by hand or via a tool".
var devRuntimes = map[string]bool{
	"node": true, "bun": true, "deno": true,
	"python": true, "python3": true, "ruby": true, "php": true,
	"java": true, "go": true, "cargo": true, "dotnet": true,
	"beam.smp": true, "erl": true, "next-server": true,
	"vite": true, "webpack": true, "gunicorn": true, "uvicorn": true,
	"flask": true, "rails": true, "puma": true, "caddy": true,
}

var systemPathPrefixes = []string{
	"/System/", "/usr/libexec/", "/usr/sbin/", "/Library/Apple/",
	"/usr/lib/systemd/", "/lib/systemd/",
}

// AssessSafety classifies one listener: is it safe to send off, and
// what actually happens if you do. Heuristics, honest ones: name tables
// for known daemons, then binary path, then owner, then app bundles,
// then origin and cwd.
func AssessSafety(d *Detail, env SafetyEnv) Safety {
	name := d.Process.Name
	cmd := d.Process.Cmdline

	if what, ok := systemDaemons[name]; ok {
		return level(SafetySystem,
			"macOS system daemon",
			what+" would drop; launchd relights it on its own")
	}

	// Docker's backend masquerades as an app helper but takes the
	// whole container runtime with it.
	if strings.HasPrefix(name, "com.docker.") || name == "Docker Desktop" {
		return level(SafetyCaution,
			"Docker Desktop backend",
			"every running container and published port goes down with it")
	}

	exe := firstField(cmd)
	for _, p := range systemPathPrefixes {
		if strings.HasPrefix(exe, p) {
			return level(SafetySystem,
				"OS binary in "+p,
				"the feature it backs may hiccup; launchd usually relights it")
		}
	}

	owner := d.Process.User
	if owner == "root" || strings.HasPrefix(owner, "_") {
		return level(SafetySystem,
			"runs as "+owner,
			"system-level service: sending it off can break OS features and may need sudo")
	}

	// A known dev runtime outranks the .app check: Homebrew's python
	// runs from Python.framework/…/Python.app but is still a dev server.
	cwd := d.Process.Cwd
	inHome := env.Home != "" && strings.HasPrefix(cwd, env.Home) &&
		!strings.HasPrefix(cwd, env.Home+"/Library")
	if devRuntimes[runtimeName(name)] {
		if d.Origin != "manual" {
			return level(SafetySafe,
				"lit by "+d.Origin,
				"the port frees up; "+d.Origin+" can relight it when needed")
		}
		reason := "dev server"
		if cwd != "" && cwd != "/" {
			reason = "dev server in " + tildePath(cwd, env.Home)
		}
		return level(SafetySafe,
			reason,
			"only this server stops; the port frees up and you can restart it any time")
	}

	if app := appBundle(cmd); app != "" {
		return level(SafetyCaution,
			"part of "+app+".app",
			app+" may lose this feature or crash; apps usually relaunch their helpers")
	}
	if strings.Contains(name, "Helper") {
		return level(SafetyCaution,
			"helper of a desktop app",
			"its parent app may misbehave until it relaunches the helper")
	}

	if d.Origin != "manual" {
		return level(SafetySafe,
			"lit by "+d.Origin,
			"the port frees up; "+d.Origin+" can relight it when needed")
	}

	if inHome {
		return level(SafetySafe,
			"your process, working in "+tildePath(cwd, env.Home),
			"nothing else depends on it; restart it whenever you like")
	}

	return level(SafetyUnknown,
		"unrecognised process",
		"open the detail panel and check its command before sending it off")
}

func level(l SafetyLevel, reason, consequence string) Safety {
	return Safety{Level: l, Label: l.String(), Reason: reason, Consequence: consequence}
}

// appBundle extracts "Discord" from ".../Discord.app/Contents/...".
// Bundles buried inside a .framework are runtimes, not desktop apps.
func appBundle(cmdline string) string {
	head, _, found := strings.Cut(cmdline, ".app/")
	if !found || strings.Contains(head, ".framework/") {
		return ""
	}
	if j := strings.LastIndexByte(head, '/'); j >= 0 {
		head = head[j+1:]
	}
	return head
}

// runtimeName normalises "python3.12" → "python3", "node (v8)" → "node".
func runtimeName(name string) string {
	n := strings.ToLower(baseName(name))
	if i := strings.IndexByte(n, ' '); i >= 0 {
		n = n[:i]
	}
	n = strings.TrimRight(n, "0123456789.")
	if devRuntimes[n] {
		return n
	}
	// keep the trailing digits variant ("python3") if the bare stem missed
	full := strings.ToLower(baseName(name))
	if i := strings.IndexByte(full, ' '); i >= 0 {
		full = full[:i]
	}
	return full
}

func firstField(s string) string {
	if f := strings.Fields(s); len(f) > 0 {
		return f[0]
	}
	return ""
}

// tildePath shortens /Users/you/x to ~/x for display.
func tildePath(p, home string) string {
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
