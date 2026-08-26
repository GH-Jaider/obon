// Package ops performs the send-off: terminating processes and reporting
// whether their ports are actually free again.
package ops

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"syscall"
	"time"
)

// Result is the outcome of sending off one PID.
type Result struct {
	PID      int32  `json:"pid"`
	Name     string `json:"name,omitempty"`
	Port     int    `json:"port"`
	Signaled string `json:"signaled"` // "term" | "kill"
	Exited   bool   `json:"exited"`
	PortFree bool   `json:"port_free"`
	Err      error  `json:"-"`
}

// Target is one process selected for a send-off.
type Target struct {
	PID  int32
	Name string
	Port int
}

// SendOff terminates targets with SIGTERM, escalating to SIGKILL after
// grace for survivors. It never returns early on individual failures.
func SendOff(ctx context.Context, targets []Target, grace time.Duration) []Result {
	results := make([]Result, len(targets))
	for i, t := range targets {
		results[i] = Result{PID: t.PID, Name: t.Name, Port: t.Port}
		if err := signal(t.PID, syscall.SIGTERM); err != nil {
			results[i].Err = err
			continue
		}
		results[i].Signaled = "term"
	}

	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if allDone(results) {
			break
		}
		time.Sleep(80 * time.Millisecond)
	}
	for i := range results {
		r := &results[i]
		if r.Err != nil || r.Exited {
			continue
		}
		if err := signal(r.PID, syscall.SIGKILL); err != nil {
			if !alive(r.PID) {
				r.Exited = true // died between checks
			} else {
				r.Err = fmt.Errorf("escalation failed: %w", err)
			}
			continue
		}
		r.Signaled = "kill"
	}

	for i := range results {
		r := &results[i]
		r.Exited = r.Exited || (!alive(r.PID))
		if r.Err == nil && r.Port > 0 {
			r.PortFree = portFree(r.Port)
		}
	}
	return results
}

func allDone(rs []Result) bool {
	for _, r := range rs {
		if r.Err == nil && !r.Exited && alive(r.PID) {
			return false
		}
	}
	return true
}

func signal(pid int32, sig syscall.Signal) error {
	// syscall.Kill takes int; on Windows only Kill works via taskkill-like paths.
	p := int(pid)
	if p <= 0 {
		return fmt.Errorf("invalid pid %d", pid)
	}
	return syscall.Kill(p, sig)
}

func alive(pid int32) bool {
	if pid <= 0 {
		return false
	}
	if err := syscall.Kill(int(pid), 0); err != nil {
		return false
	}
	return true
}

// portFree probes loopback TCP; UDP and non-loopback binds report true
// optimistically after the owner exited.
func portFree(port int) bool {
	c, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 150*time.Millisecond)
	if err != nil {
		return true // refused => nobody listening
	}
	_ = c.Close()
	return false
}
