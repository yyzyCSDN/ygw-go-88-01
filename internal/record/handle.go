package record

import (
	"os"
	"path/filepath"
)

func openLog(dir, name string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

func closeLog(f *os.File) error {
	if f == nil {
		return nil
	}
	return f.Close()
}

// closeLogFn is the indirection used by Append so tests can force a close error
// after a successful Sync and assert the seq still advances (no duplicate).
var closeLogFn = closeLog

// writeFileAtomic writes data to path atomically: it writes to a temp file in
// the same directory, fsyncs, then renames over the target. A crash mid-write
// therefore cannot leave a truncated or partial counter behind.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// writeCounter persists a counter value to <dir>/<name> atomically.
func writeCounter(dir, name string, data []byte) error {
	return writeFileAtomic(filepath.Join(dir, name), data, 0644)
}

// readCounter reads the counter value previously written by writeCounter. The
// second result is false when no counter file exists yet.
func readCounter(dir, name string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return "", false
	}
	return string(data), true
}
