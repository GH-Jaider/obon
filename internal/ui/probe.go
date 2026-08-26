package ui

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// probeResult is what answered (or didn't) on http://localhost:port.
type probeResult struct {
	status int
	ctype  string
	title  string
	err    error
}

type probedMsg struct {
	key string // group key the probe belongs to
	res probeResult
}

type openedMsg struct{ url string }

// probeCmd asks the port what it is serving, briefly and once.
func probeCmd(key string, port int) tea.Cmd {
	return func() tea.Msg {
		return probedMsg{key: key, res: probePort(port)}
	}
}

func probePort(port int) probeResult {
	client := &http.Client{
		Timeout: 900 * time.Millisecond,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse // report the redirect itself
		},
	}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
	if err != nil {
		return probeResult{err: err}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	return probeResult{
		status: resp.StatusCode,
		ctype:  cleanCtype(resp.Header.Get("Content-Type")),
		title:  htmlTitle(body),
	}
}

func cleanCtype(ct string) string {
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return strings.TrimSpace(ct)
}

var titleRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

func htmlTitle(body []byte) string {
	m := titleRe.FindSubmatch(body)
	if m == nil {
		return ""
	}
	return strings.Join(strings.Fields(string(m[1])), " ")
}

// openBrowserCmd hands the port's URL to the OS browser.
func openBrowserCmd(port int) tea.Cmd {
	url := localURL(port)
	return func() tea.Msg {
		var c *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			c = exec.Command("open", url)
		case "windows":
			c = exec.Command("cmd", "/c", "start", url)
		default:
			c = exec.Command("xdg-open", url)
		}
		_ = c.Start()
		return openedMsg{url: url}
	}
}

func localURL(port int) string {
	return fmt.Sprintf("http://localhost:%d/", port)
}

// hyperlink wraps text in an OSC 8 sequence so terminals that support
// it (iTerm2, Ghostty, WezTerm, Kitty…) make it clickable.
func hyperlink(url, text string) string {
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}
