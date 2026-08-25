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
	f, err := openLog(j.dir, "journal.log")
	if err != nil {
		return err
	}
	defer closeLog(f)
	sum := xxhash.Sum64String(reactorID + event)
	line := fmt.Sprintf("%s %s %s %d\n", j.now().UTC().Format(time.RFC3339), reactorID, event, sum)
	_, err = f.WriteString(line)
	return err
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
