package record

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"testing"

	"chemicalprocessdcs/internal/model"
)

// errSentinel is the synthetic close error injected by the close-failure tests.
var errSentinel = errors.New("record: forced close failure")

func mustAppend(t *testing.T, s *Service, id string, temp, pres float64) {
	t.Helper()
	if err := s.Append(model.RecordEntry{ReactorID: id, State: model.ReactorHeating, Temperature: temp, Pressure: pres}); err != nil {
		t.Fatalf("Append: %v", err)
	}
}

// TestAppendConcurrentContiguous proves the core fix: many goroutines appending
// concurrently produce a strictly increasing, gap-free, duplicate-free sequence
// whose persisted counter equals the number of lines written.
func TestAppendConcurrentContiguous(t *testing.T) {
	dir := t.TempDir()
	s := NewService(dir)

	const reactors = 20
	const perReactor = 100
	var wg sync.WaitGroup
	wg.Add(reactors)
	for r := 0; r < reactors; r++ {
		go func(id int) {
			defer wg.Done()
			rid := "R" + strconv.Itoa(id)
			for i := 0; i < perReactor; i++ {
				if err := s.Append(model.RecordEntry{
					ReactorID:   rid,
					State:       model.ReactorHeating,
					Temperature: 100 + float64(i),
					Pressure:    5.0,
				}); err != nil {
					t.Errorf("Append: %v", err)
					return
				}
			}
		}(r)
	}
	wg.Wait()

	want := uint64(reactors * perReactor)
	if got := s.Seq(); got != want {
		t.Fatalf("Seq = %d, want %d", got, want)
	}

	entries, err := s.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if uint64(len(entries)) != want {
		t.Fatalf("lines = %d, want %d", len(entries), want)
	}

	seqs := make([]uint64, 0, len(entries))
	for _, e := range entries {
		seqs = append(seqs, e.Seq)
	}
	sort.Slice(seqs, func(i, j int) bool { return seqs[i] < seqs[j] })
	for i, sq := range seqs {
		if sq != uint64(i+1) {
			t.Fatalf("seq at sorted pos %d = %d, want %d (gap or duplicate)", i, sq, i+1)
		}
	}

	// Persisted counter must match the last written seq.
	raw, ok := readCounter(dir, runCounterFile)
	if !ok {
		t.Fatal("run.counter missing")
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || n != want {
		t.Fatalf("run.counter = %v, want %d", raw, want)
	}
}

// TestSeqSurvivesRestart proves the counter is loaded from disk so the sequence
// continues instead of resetting to 0 and colliding with prior entries.
func TestSeqSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	s := NewService(dir)
	mustAppend(t, s, "R1", 100, 5)
	mustAppend(t, s, "R1", 110, 5)
	mustAppend(t, s, "R1", 120, 5)
	if got := s.Seq(); got != 3 {
		t.Fatalf("Seq = %d, want 3", got)
	}

	// Simulate a restart by constructing a fresh Service over the same dir.
	s2 := NewService(dir)
	if got := s2.Seq(); got != 3 {
		t.Fatalf("after restart Seq = %d, want 3 (counter not loaded)", got)
	}
	mustAppend(t, s2, "R1", 130, 5)
	if got := s2.Seq(); got != 4 {
		t.Fatalf("after restart+append Seq = %d, want 4 (seq reused/collided)", got)
	}

	entries, err := s2.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("lines = %d, want 4", len(entries))
	}
	last := entries[len(entries)-1]
	if last.Seq != 4 {
		t.Fatalf("last seq = %d, want 4", last.Seq)
	}
}

// TestAppendFailureDoesNotConsumeSeq proves a failed write does not advance the
// counter: retrying the same entry after a transient failure yields a contiguous
// sequence with no gap and no duplicate.
func TestAppendFailureDoesNotConsumeSeq(t *testing.T) {
	dir := t.TempDir()
	s := NewService(dir)
	mustAppend(t, s, "R1", 100, 5) // seq 1
	mustAppend(t, s, "R1", 110, 5) // seq 2

	// Force openLog to fail by pointing the service at a path whose parent is a
	// regular file (MkdirAll cannot create a directory under a file).
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	badDir := filepath.Join(blocker, "nested")
	goodDir := s.dir
	s.dir = badDir

	entry := model.RecordEntry{ReactorID: "R1", State: model.ReactorHeating, Temperature: 120, Pressure: 5}
	if err := s.Append(entry); err == nil {
		t.Fatal("Append to bad dir: want error, got nil")
	}
	if got := s.Seq(); got != 2 {
		t.Fatalf("after failed Append Seq = %d, want 2 (seq consumed by a failed write)", got)
	}

	// Restore the good dir and retry the same entry: it must take seq 3, leaving
	// the log contiguous (1,2,3) with no gap and no duplicate.
	s.dir = goodDir
	mustAppend(t, s, "R1", 120, 5)
	if got := s.Seq(); got != 3 {
		t.Fatalf("after retry Seq = %d, want 3", got)
	}

	entries, err := s.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	wantSeqs := []uint64{1, 2, 3}
	for i, e := range entries {
		if e.Seq != wantSeqs[i] {
			t.Fatalf("entry %d seq = %d, want %d", i, e.Seq, wantSeqs[i])
		}
	}
}

// TestTrimKeepsSequenceMonotonic proves Trim (which rewrites run.log) is
// serialized against Append and never causes a seq to be reused: after trimming,
// the next append continues from the high-water mark rather than restarting.
func TestTrimKeepsSequenceMonotonic(t *testing.T) {
	dir := t.TempDir()
	s := NewService(dir)
	for i := 0; i < 10; i++ {
		mustAppend(t, s, "R1", 100+float64(i), 5)
	}
	if got := s.Seq(); got != 10 {
		t.Fatalf("Seq = %d, want 10", got)
	}

	if err := s.Trim(5); err != nil {
		t.Fatalf("Trim: %v", err)
	}
	entries, err := s.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("after Trim lines = %d, want 5", len(entries))
	}
	// Trimmed entries keep their original (high) seqs.
	for _, e := range entries {
		if e.Seq < 6 || e.Seq > 10 {
			t.Fatalf("trimmed seq = %d, want in [6,10]", e.Seq)
		}
	}
	if got := s.Seq(); got != 10 {
		t.Fatalf("after Trim Seq = %d, want 10 (counter regressed)", got)
	}

	// Next append must continue past the high-water mark, not reuse a trimmed seq.
	mustAppend(t, s, "R1", 200, 5)
	if got := s.Seq(); got != 11 {
		t.Fatalf("after Trim+append Seq = %d, want 11 (seq reused)", got)
	}
}

// TestJournalConcurrentContiguous proves the journal path is serialized and its
// counter is persisted, mirroring the run-log guarantees.
func TestJournalConcurrentContiguous(t *testing.T) {
	dir := t.TempDir()
	j := NewJournal(dir)

	const writers = 20
	const perWriter = 100
	var wg sync.WaitGroup
	wg.Add(writers)
	for w := 0; w < writers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				if err := j.Append("R"+strconv.Itoa(id), "tick"); err != nil {
					t.Errorf("Append: %v", err)
					return
				}
			}
		}(w)
	}
	wg.Wait()

	want := writers * perWriter
	if got := j.PersistedCount(); got != want {
		t.Fatalf("PersistedCount = %d, want %d", got, want)
	}
	raw, ok := readCounter(dir, journalCounter)
	if !ok {
		t.Fatal("journal.counter missing")
	}
	if n, _ := strconv.Atoi(raw); n != want {
		t.Fatalf("journal.counter = %s, want %d", raw, want)
	}

	// Restart: counter survives.
	j2 := NewJournal(dir)
	if got := j2.PersistedCount(); got != want {
		t.Fatalf("after restart PersistedCount = %d, want %d", got, want)
	}
}

// TestJournalPersistedCountReconciles proves a stale counter file is reconciled
// against the actual line count on startup so it never drifts.
func TestJournalPersistedCountReconciles(t *testing.T) {
	dir := t.TempDir()
	j := NewJournal(dir)
	for i := 0; i < 5; i++ {
		if err := j.Append("R1", "tick"); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	// Corrupt the counter file to a stale, lower value.
	if err := os.WriteFile(filepath.Join(dir, journalCounter), []byte("2"), 0644); err != nil {
		t.Fatalf("write counter: %v", err)
	}
	j2 := NewJournal(dir)
	if got := j2.PersistedCount(); got != 5 {
		t.Fatalf("reconciled PersistedCount = %d, want 5", got)
	}
}

// TestServiceCounterReconciles proves the run-log counter is reconciled against
// the highest seq actually present in the log, so a stale counter file cannot
// cause a seq reuse.
func TestServiceCounterReconciles(t *testing.T) {
	dir := t.TempDir()
	s := NewService(dir)
	for i := 0; i < 7; i++ {
		mustAppend(t, s, "R1", 100+float64(i), 5)
	}
	// Stale the counter below the real high-water mark.
	if err := os.WriteFile(filepath.Join(dir, runCounterFile), []byte("3"), 0644); err != nil {
		t.Fatalf("write counter: %v", err)
	}
	s2 := NewService(dir)
	if got := s2.Seq(); got != 7 {
		t.Fatalf("reconciled Seq = %d, want 7", got)
	}
	// Next append continues monotonically; no reuse of 4..7.
	mustAppend(t, s2, "R1", 200, 5)
	if got := s2.Seq(); got != 8 {
		t.Fatalf("after stale+append Seq = %d, want 8", got)
	}
}

// TestAppendCloseErrorDoesNotReuseSeq proves the close-after-Sync fix: once the
// line is durably written (Sync ok), a close error must still advance the seq.
// Otherwise a retry of the same entry would reuse the seq and write a duplicate.
func TestAppendCloseErrorDoesNotReuseSeq(t *testing.T) {
	dir := t.TempDir()
	s := NewService(dir)
	mustAppend(t, s, "R1", 100, 5) // seq 1
	mustAppend(t, s, "R1", 110, 5) // seq 2

	// Force the next close to fail after a successful Sync. The line is durable.
	prev := closeLogFn
	var injected bool
	closeLogFn = func(f *os.File) error {
		err := f.Close() // release the fd; the data is already synced to disk
		if !injected {
			injected = true
			return errSentinel // surface the close error on this one append
		}
		return err
	}
	t.Cleanup(func() { closeLogFn = prev })

	entry := model.RecordEntry{ReactorID: "R1", State: model.ReactorHeating, Temperature: 120, Pressure: 5}
	if err := s.Append(entry); err != errSentinel {
		t.Fatalf("Append error = %v, want errSentinel (close failure surfaced)", err)
	}
	// Despite the surfaced close error, the durable line took seq 3: the next
	// append must take seq 4, never reusing 3 (which would duplicate it).
	if got := s.Seq(); got != 3 {
		t.Fatalf("after close-failed Append Seq = %d, want 3 (seq not advanced despite durable write)", got)
	}
	mustAppend(t, s, "R1", 130, 5)
	if got := s.Seq(); got != 4 {
		t.Fatalf("after retry Seq = %d, want 4", got)
	}

	entries, err := s.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	// Exactly one line per seq: no duplicate of seq 3.
	seen := make(map[uint64]int, len(entries))
	for _, e := range entries {
		seen[e.Seq]++
	}
	for sq, n := range seen {
		if n != 1 {
			t.Fatalf("seq %d appears %d times (duplicate after close failure)", sq, n)
		}
	}
}

// TestJournalAppendCloseErrorStillCounts proves the journal path mirrors the
// run-log fix: a close error after a durable write still advances the count.
func TestJournalAppendCloseErrorStillCounts(t *testing.T) {
	dir := t.TempDir()
	j := NewJournal(dir)
	for i := 0; i < 3; i++ {
		if err := j.Append("R1", "tick"); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	prev := closeLogFn
	var injected bool
	closeLogFn = func(f *os.File) error {
		err := f.Close()
		if !injected {
			injected = true
			return errSentinel
		}
		return err
	}
	t.Cleanup(func() { closeLogFn = prev })

	if err := j.Append("R1", "tick"); err != errSentinel {
		t.Fatalf("Append error = %v, want errSentinel", err)
	}
	if got := j.PersistedCount(); got != 4 {
		t.Fatalf("PersistedCount = %d, want 4 (count not advanced after durable write)", got)
	}
	// A fresh instance reconciles against the actual line count.
	j2 := NewJournal(dir)
	if got := j2.PersistedCount(); got != 4 {
		t.Fatalf("after restart PersistedCount = %d, want 4", got)
	}
}
