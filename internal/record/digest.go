package record

import (
	"strings"

	"chemicalprocessdcs/internal/model"

	"github.com/cespare/xxhash/v2"
)

func DigestOf(entries []model.RecordEntry) uint64 {
	if len(entries) == 0 {
		return 0
	}
	var builder strings.Builder
	for _, entry := range entries {
		builder.WriteString(entry.ReactorID)
		builder.WriteString(string(entry.State))
		builder.WriteString("|")
	}
	return xxhash.Sum64String(builder.String())
}

func (s *Service) VerifyDigest() (uint64, error) {
	entries, err := s.ReadAll()
	if err != nil {
		return 0, err
	}
	return DigestOf(entries), nil
}
