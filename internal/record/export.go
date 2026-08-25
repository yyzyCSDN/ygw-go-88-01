package record

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func (j *Journal) ReadAll() ([]string, error) {
	f, err := os.Open(filepath.Join(j.dir, "journal.log"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			lines = append(lines, text)
		}
	}
	return lines, scanner.Err()
}
