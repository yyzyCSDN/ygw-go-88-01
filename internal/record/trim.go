package record

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cespare/xxhash/v2"
)

// Trim truncates the run log to the most recent maxEntries records. It is
// serialized against Append so the rewrite never races an in-flight write.
// The seq counter is left untouched (seq is monotonic and never reused), but
// the run.counter file is rewritten to match the trimmed high-water mark so a
// restart keeps the counter consistent with the trimmed log.
func (s *Service) Trim(maxEntries int) error {
	if maxEntries < 0 {
		maxEntries = 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.ReadAll()
	if err != nil {
		return err
	}
	if len(entries) <= maxEntries {
		return nil
	}
	entries = entries[len(entries)-maxEntries:]

	// Write the trimmed log via a temp file in the same dir, then rename over
	// run.log so a crash mid-trim cannot leave a partial/truncated log.
	f, err := os.CreateTemp(s.dir, ".trim-*")
	if err != nil {
		return err
	}
	tmpName := f.Name()
	for _, entry := range entries {
		sum := xxhash.Sum64String(entry.ReactorID)
		line := fmt.Sprintf("%d %s %s %.2f %.2f %d\n", entry.Seq, entry.ReactorID, entry.State, entry.Temperature, entry.Pressure, sum)
		if _, err := f.WriteString(line); err != nil {
			f.Close()
			os.Remove(tmpName)
			return err
		}
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpName)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, filepath.Join(s.dir, runLogFile)); err != nil {
		os.Remove(tmpName)
		return err
	}

	// Reconcile the in-memory high-water mark with the trimmed log so a stale
	// counter can never cause a seq reuse after restart. seq is monotonic, so
	// we only ever raise it.
	for _, entry := range entries {
		if entry.Seq > s.seq {
			s.seq = entry.Seq
		}
	}
	return s.persistCounterLocked()
}
