package record

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"chemicalprocessdcs/internal/model"
)

func (s *Service) ReadAll() ([]model.RecordEntry, error) {
	f, err := os.Open(filepath.Join(s.dir, "run.log"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []model.RecordEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 6 {
			continue
		}
		seq, _ := strconv.ParseUint(fields[0], 10, 64)
		temp, _ := strconv.ParseFloat(fields[3], 64)
		pressure, _ := strconv.ParseFloat(fields[4], 64)
		entries = append(entries, model.RecordEntry{
			Seq:         seq,
			ReactorID:   fields[1],
			State:       model.ReactorState(fields[2]),
			Temperature: temp,
			Pressure:    pressure,
		})
	}
	return entries, scanner.Err()
}
