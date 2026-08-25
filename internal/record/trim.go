package record

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cespare/xxhash/v2"
)

func (s *Service) Trim(maxEntries int) error {
	entries, err := s.ReadAll()
	if err != nil {
		return err
	}
	if len(entries) <= maxEntries {
		return nil
	}
	entries = entries[len(entries)-maxEntries:]
	f, err := os.Create(filepath.Join(s.dir, "run.log"))
	if err != nil {
		return err
	}
	defer f.Close()
	for _, entry := range entries {
		sum := xxhash.Sum64String(entry.ReactorID)
		line := fmt.Sprintf("%d %s %s %.2f %.2f %d\n", entry.Seq, entry.ReactorID, entry.State, entry.Temperature, entry.Pressure, sum)
		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}
	return nil
}
