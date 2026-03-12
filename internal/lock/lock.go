package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// Lock provides single-instance safety via flock.
type Lock struct {
	path string
	f    *os.File
}

// Acquire creates an exclusive lock on LOCK in storeDir.
// Returns an error if another instance already holds the lock.
func Acquire(storeDir string) (*Lock, error) {
	if err := os.MkdirAll(storeDir, 0700); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}

	lockPath := filepath.Join(storeDir, "LOCK")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		// Read existing PID for error message.
		data, _ := os.ReadFile(lockPath)
		f.Close()
		return nil, fmt.Errorf("another tgcli instance is running (lock held by PID %s)", string(data))
	}

	// Write PID + timestamp.
	_ = f.Truncate(0)
	_, _ = f.Seek(0, 0)
	_, _ = fmt.Fprintf(f, "%d %s", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
	_ = f.Sync()

	return &Lock{path: lockPath, f: f}, nil
}

// Release releases the lock.
func (l *Lock) Release() error {
	if l.f == nil {
		return nil
	}
	_ = syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	err := l.f.Close()
	l.f = nil
	return err
}

// Info returns PID info from the lock file (for diagnostics).
func Info(storeDir string) (pid int, ts time.Time, err error) {
	lockPath := filepath.Join(storeDir, "LOCK")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return 0, time.Time{}, err
	}
	parts := string(data)
	if len(parts) > 0 {
		// Parse "PID TIMESTAMP"
		var pidStr, tsStr string
		n, _ := fmt.Sscanf(parts, "%s %s", &pidStr, &tsStr)
		if n >= 1 {
			pid, _ = strconv.Atoi(pidStr)
		}
		if n >= 2 {
			ts, _ = time.Parse(time.RFC3339, tsStr)
		}
	}
	return pid, ts, nil
}
