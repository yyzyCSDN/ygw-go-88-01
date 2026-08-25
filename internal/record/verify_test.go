package record_test

import (
	"fmt"
	"sync"
	"testing"

	"chemicalprocessdcs/internal/record"
)

func TestJournalConcurrentAppend(t *testing.T) {
	j := record.NewJournal(t.TempDir())
	const n = 500
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = j.Append("R1", fmt.Sprintf("event-%d", i))
		}(i)
	}
	wg.Wait()
	lines, err := j.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != n {
		t.Fatalf("journal lost entries: got %d want %d", len(lines), n)
	}
	if j.PersistedCount() == 0 {
		t.Fatal("journal counter must be persisted")
	}
}
