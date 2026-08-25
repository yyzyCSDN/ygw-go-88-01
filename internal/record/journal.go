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

// Journal serializes concurrent journal-log writes. See Service for the
// rationale: the append path is serialized so the counter matches the last
// written line and a failed write does not consume a counter.
type Journal struct {
	mu  sync.Mutex
	dir string
	// total is the persisted line count; loaded from journal.counter at startup.
	total uint64
	// writes counts accepted writes since startup (diagnostics only).
	writes uint64
	now    func() time.Time
}

const (
	journalLogFile = "journal.log"
	journalCounter = "journal.counter"
)

// NewJournal opens (or creates) the journal directory and resumes the count
// from disk, reconciling with the line count actually in the file.
func NewJournal(dir string) *Journal {
	j := &Journal{dir: dir, now: time.Now}
	j.total = j.loadCounter()
	return j
}

func (j *Journal) loadCounter() uint64 {
	var counter uint64
	if raw, ok := readCounter(j.dir, journalCounter); ok {
		if n, err := strconv.ParseUint(raw, 10, 64); err == nil {
			counter = n
		}
	}
	lines, err := j.ReadAll()
	if err != nil {
		return counter
	}
	if uint64(len(lines)) > counter {
		counter = uint64(len(lines))
	}
	return counter
}

func (j *Journal) persistCounterLocked() error {
	return writeCounter(j.dir, journalCounter, []byte(strconv.FormatUint(j.total, 10)))
}

// Append writes one journal line. The whole operation is serialized: the
// counter is advanced only after the line is durably written, then persisted.
// A failed write does not consume the counter, so the caller may retry the
// same event without producing a gap or duplicate.
func (j *Journal) Append(reactorID string, event string) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	f, err := openLog(j.dir, journalLogFile)
	if err != nil {
		return err
	}
	sum := xxhash.Sum64String(reactorID + event)
	line := fmt.Sprintf("%s %s %s %d\n", j.now().UTC().Format(time.RFC3339), reactorID, event, sum)
	if _, err = f.WriteString(line); err != nil {
		_ = closeLog(f)
		return err
	}
	if err = f.Sync(); err != nil {
		_ = closeLog(f)
		return err
	}
	// Sync succeeded, so the line is durable. A close error past this point is
	// a resource-release error, not data loss: the count still advances so a
	// retry never double-counts. Surface the close error only after committing.
	closeErr := closeLogFn(f)

	j.total++
	atomic.AddUint64(&j.writes, 1)
	_ = j.persistCounterLocked()
	return closeErr
}

// PersistedCount returns the count of journal lines persisted to disk. It is
// the in-memory high-water mark, which is kept in sync with journal.counter.
func (j *Journal) PersistedCount() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return int(j.total)
}

// Total is an alias for PersistedCount kept for clarity.
func (j *Journal) Total() uint64 {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.total
}

func (j *Journal) AppendEvent(ev model.Event) error {
	return j.Append(ev.ReactorID, ev.Kind+" "+ev.Message)
}
