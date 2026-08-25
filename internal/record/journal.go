package record

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"chemicalprocessdcs/internal/model"

	"github.com/cespare/xxhash/v2"
)

type Journal struct {
	mu      sync.Mutex
	dir     string
	handles int
	total   int
	now     func() time.Time
}

func NewJournal(dir string) *Journal {
	return &Journal{dir: dir, now: time.Now}
}

func (j *Journal) Append(reactorID string, event string) error {
	return j.appendWithRetry(reactorID, event, 3)
}

func (j *Journal) appendWithRetry(reactorID string, event string, attempts int) error {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := j.appendOnce(reactorID, event); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return lastErr
}

func (j *Journal) appendOnce(reactorID string, event string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	f, err := openLog(j.dir, "journal.log")
	if err != nil {
		return err
	}
	j.handles++
	j.total++
	defer func() {
		_ = closeLog(f)
		j.handles--
	}()
	sum := xxhash.Sum64String(reactorID + event)
	line := fmt.Sprintf("%s %s %s %d\n", j.now().UTC().Format(time.RFC3339), reactorID, event, sum)
	if _, err := f.WriteString(line); err != nil {
		return err
	}
	return j.flushCounterLocked()
}

func (j *Journal) flushCounterLocked() error {
	return os.WriteFile(filepath.Join(j.dir, "journal.counter"), []byte(fmt.Sprintf("%d\n", j.total)), 0644)
}

func (j *Journal) PersistedCount() int {
	data, err := os.ReadFile(filepath.Join(j.dir, "journal.counter"))
	if err != nil {
		return 0
	}
	count, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return count
}

func (j *Journal) AppendEvent(ev model.Event) error {
	return j.Append(ev.ReactorID, ev.Kind+" "+ev.Message)
}
