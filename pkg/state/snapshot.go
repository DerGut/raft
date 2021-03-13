package state

import (
	"encoding/json"
)

type snapshot struct {
	LastIncludedIndex int    `json`
	LastIncludedTerm  Term   `json`
	ApplicationState  []byte `json`
}

func (s *snapshot) Write(dirpath string) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}

	return writeToTimestampedFile(dirpath, b)
}

func deleteOldSnapshots(dirpath string) error {
	return deleteOldFiles(dirpath)
}
