package testharness

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Harness owns isolated filesystem and environment state for a contract test.
type Harness struct {
	Root        string
	ProjectID   string
	SessionID   string
	TaskData    string
	CacheDir    string
	BinDir      string
	Environment []string
}

// NewHarness creates isolated project, session, cache, and executable state.
func NewHarness(t *testing.T, projectID string, sessionID string) *Harness {
	t.Helper()

	root := t.TempDir()
	harness := &Harness{
		Root:      root,
		ProjectID: projectID,
		SessionID: sessionID,
		TaskData:  filepath.Join(root, "taskdata"),
		CacheDir:  filepath.Join(root, "cache"),
		BinDir:    filepath.Join(root, "bin"),
	}

	for _, path := range []string{
		harness.TaskData,
		harness.CacheDir,
		harness.BinDir,
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("create isolated harness directory %q: %v", path, err)
		}
	}

	t.Setenv("HOME", root)
	t.Setenv("PROJECT_ID", projectID)
	t.Setenv("TASKDATA", harness.TaskData)
	t.Setenv("JAFLOW_CACHE_DIR", harness.CacheDir)
	t.Setenv("JACAZUL_SESSION_ID", sessionID)
	t.Setenv("JACAZUL_HOME", filepath.Join(root, ".jacazul-ai"))

	harness.Environment = os.Environ()
	harness.Environment = append(
		harness.Environment,
		"PROJECT_ID="+projectID,
		"TASKDATA="+harness.TaskData,
		"JAFLOW_CACHE_DIR="+harness.CacheDir,
		"JACAZUL_SESSION_ID="+sessionID,
		"JACAZUL_HOME="+filepath.Join(root, ".jacazul-ai"),
	)

	return harness
}

// WriteFile writes a file inside the harness and creates its parent directory.
func (h *Harness) WriteFile(t *testing.T, relativePath string, content []byte) string {
	t.Helper()

	path := filepath.Join(h.Root, filepath.Clean(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent directory for %q: %v", relativePath, err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write isolated file %q: %v", relativePath, err)
	}
	return path
}

// FakeCommand installs a deterministic executable at the front of PATH.
func (h *Harness) FakeCommand(t *testing.T, name string, script string) string {
	t.Helper()

	path := filepath.Join(h.BinDir, name)
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake command %q: %v", name, err)
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", h.BinDir+string(os.PathListSeparator)+oldPath)
	h.Environment = append(h.Environment, "PATH="+os.Getenv("PATH"))
	return path
}

// RunCommand executes a command with the harness environment and captures all
// output for assertions.
func (h *Harness) RunCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = h.Environment
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("run %s: %w: %s", name, err, output)
	}
	return string(output), nil
}
