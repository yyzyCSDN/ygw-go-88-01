package record

import (
	"fmt"
	"sync"

	"chemicalprocessdcs/internal/model"

	"github.com/cespare/xxhash/v2"
)

type Service struct {
	mu      sync.Mutex
	dir     string
	seq     uint64
	handles int
}

func NewService(dir string) *Service {
	return &Service{dir: dir}
}

func (s *Service) Append(entry model.RecordEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	entry.Seq = s.seq
	f, err := openLog(s.dir, "run.log")
	if err != nil {
		return err
	}
	s.handles++
	defer func() {
		_ = closeLog(f)
		s.handles--
	}()
	sum := xxhash.Sum64String(entry.ReactorID)
	line := fmt.Sprintf("%d %s %s %.2f %.2f %d\n", entry.Seq, entry.ReactorID, entry.State, entry.Temperature, entry.Pressure, sum)
	_, err = f.WriteString(line)
	return err
}

