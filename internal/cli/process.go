package cli

import (
	"fmt"
	"time"

	"github.com/padovanl/portop/internal/procctl"
	"github.com/padovanl/portop/internal/procinfo"
)

// signalProcess checks the process start time before sending a signal. A PID
// can be reused after the row was displayed, so the PID alone is not enough.
func signalProcess(pid int, expectedStart string, force bool) error {
	if pid <= 0 || expectedStart == "" {
		return fmt.Errorf("a PID and process start time are required")
	}
	expected, err := time.Parse(time.RFC3339Nano, expectedStart)
	if err != nil {
		return fmt.Errorf("invalid process start time")
	}
	info, err := procinfo.Load(pid)
	if err != nil {
		return err
	}
	if info.StartTime.IsZero() || !info.StartTime.Equal(expected) {
		return fmt.Errorf("process changed since it was inspected; refresh before trying again")
	}
	if force {
		return procctl.Kill(pid)
	}
	return procctl.Terminate(pid)
}
