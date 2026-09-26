package cli

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/padovanl/portop/internal/procinfo"
)

func TestProcessCommands(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start test process: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	pid := strconv.Itoa(cmd.Process.Pid)
	config := t.TempDir() + "/missing.yml"
	var out, errOut bytes.Buffer
	if code := Run([]string{"--config", config, "--inspect-pid", pid}, &out, &errOut); code != 0 {
		t.Fatalf("inspect exit %d: %s", code, errOut.String())
	}
	var info procinfo.Info
	if err := json.Unmarshal(out.Bytes(), &info); err != nil || info.PID != cmd.Process.Pid || info.StartTime.IsZero() {
		t.Fatalf("inspect output: %+v, %v", info, err)
	}
	start := info.StartTime.Format(time.RFC3339Nano)
	errOut.Reset()
	if code := Run([]string{"--config", config, "--signal-pid", pid, "--expected-start", info.StartTime.Add(-time.Second).Format(time.RFC3339Nano)}, &out, &errOut); code == 0 {
		t.Fatal("stale process identity accepted")
	}
	errOut.Reset()
	if code := Run([]string{"--config", config, "--signal-pid", pid, "--expected-start", start}, &out, &errOut); code != 0 {
		t.Fatalf("terminate exit %d: %s", code, errOut.String())
	}
}
