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
