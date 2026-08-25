package record

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"chemicalprocessdcs/internal/model"

	"github.com/cespare/xxhash/v2"
)

// Service serializes concurrent run-log writes from many reactor goroutines.
// The append path is fully serialized so that:
//   - sequence numbers are strictly increasing with no gaps,
//   - a failed write does not consume a sequence number (the caller may retry
//     the same entry and still produce a contiguous, gap-free log),
//   - the persisted counter on disk always matches the last line written,
//     so after a restart the sequence continues correctly instead of
//     resetting and colliding with prior entries.
type Service struct {
	mu  sync.Mutex
	dir string
	seq uint64 // in-memory high-water mark; loaded from run.counter at startup
	// handles counts currently in-flight appends; updated under mu for diagnostics.
	handles int
	// writes counts accepted writes since startup (diagnostics only).
	writes uint64
	now    func() time.Time
}

const (
	runLogFile     = "run.log"
	runCounterFile = "run.counter"
)

// NewService opens (or creates) the run-log directory and resumes the sequence
// counter from disk. The counter is reconciled with the highest seq actually
// present in the log so a stale counter can never cause a seq reuse.
func NewService(dir string) *Service {
	s := &Service{dir: dir, now: time.Now}
	s.seq = s.loadCounter()
	return s
}

// loadCounter returns max(persisted counter, max-seq-in-log). On any error or
// absence it falls back to scanning the log, then to 0.
func (s *Service) loadCounter() uint64 {
	var counter uint64
	if raw, ok := readCounter(s.dir, runCounterFile); ok {
		if n, err := strconv.ParseUint(raw, 10, 64); err == nil {
			counter = n
		}
	}
	entries, err := s.ReadAll()
	if err != nil {
		return counter
	}
	for _, e := range entries {
		if e.Seq > counter {
			counter = e.Seq
		}
	}
	return counter
}

func (s *Service) persistCounterLocked() error {
	return writeCounter(s.dir, runCounterFile, []byte(strconv.FormatUint(s.seq, 10)))
}

// Append writes one run-log entry. The whole operation is serialized: assign
// the next seq only after the log line is durably written (Sync ok), then persist
// the counter. Once the line is durable, a close error is tolerated (like a
// counter-persist failure): the seq still advances so a retry never reuses it.
// If a write fails before it is durable, the seq is left untouched so the caller
// may retry the same entry without introducing a gap or duplicate.
func (s *Service) Append(entry model.RecordEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handles++
	defer func() { s.handles-- }()

	f, err := openLog(s.dir, runLogFile)
	if err != nil {
		return err
	}
	sum := xxhash.Sum64String(entry.ReactorID)

	// Assign the next seq only once the write succeeds, so a failure does not
	// consume a seq (the caller can retry the same entry, no gap, no dup).
	next := s.seq + 1
	line := fmt.Sprintf("%d %s %s %.2f %.2f %d\n", next, entry.ReactorID, entry.State, entry.Temperature, entry.Pressure, sum)
	if _, err = f.WriteString(line); err != nil {
		_ = closeLog(f)
		return err
	}
	if err = f.Sync(); err != nil {
		_ = closeLog(f)
		return err
	}
	// Sync succeeded, so the line is durably on disk. A close error past this
	// point is a resource-release error (e.g. a delayed EIO), not a data-loss
	// error: the seq must still be committed. If we returned here without
	// advancing s.seq, a retry of the same entry would reuse this seq and write
	// a duplicate into run.log. We surface the close error only after committing
	// the counter, exactly as we tolerate a persistCounter failure (loadCounter
	// reconciles on next startup).
	closeErr := closeLogFn(f)

	// Commit: advance the in-memory high-water mark and persist it. If the
	// counter persistence fails, the line is on disk but the counter is stale;
	// loadCounter() reconciles on next startup, so no seq is ever reused.
	s.seq = next
	atomic.AddUint64(&s.writes, 1)
	_ = s.persistCounterLocked()
	return closeErr
}

// Seq returns the current high-water sequence number.
func (s *Service) Seq() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seq
}

// Handles returns the number of append operations currently in flight.
func (s *Service) Handles() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.handles
}
