package migration

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// BackupDatabase copies an existing SQLite database and its sidecar files.
func BackupDatabase(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("database path is required")
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("inspect target database: %w", err)
	}
	backup := fmt.Sprintf("%s.backup-%s", path, time.Now().UTC().Format("20060102T150405.000000000Z"))
	if err := copyFile(path, backup); err != nil {
		return "", fmt.Errorf("backup target database: %w", err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		sidecar := path + suffix
		if _, err := os.Stat(sidecar); err == nil {
			if err := copyFile(sidecar, backup+suffix); err != nil {
				return "", fmt.Errorf("backup target database %s: %w", suffix, err)
			}
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect target database %s: %w", suffix, err)
		}
	}
	return backup, nil
}

func copyFile(source string, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	return nil
}
