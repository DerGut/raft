package state

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"time"
)

const FileFormat = time.RFC3339

type Durable struct {
	dirpath string

	CurrentTerm Term    `json`
	VotedFor    *string `json`
	Log         `json`
}

func NewDurable(dirpath string) *Durable {
	file, err := latestWrite(dirpath)
	if err != nil {
		log.Println("Error reading files in path", dirpath, err, "creating new Durable")
		return &Durable{}
	}

	d, err := read(*file)
	if err != nil {
		log.Println("Error reading", *file, err, "creating new Durable")
		return &Durable{}
	}
	return d
}

func latestWrite(dirpath string) (file *string, err error) {
	files, err := ioutil.ReadDir(dirpath)
	if err != nil {
		return nil, err
	}

	times := parseFileNames(files)
	sort.Slice(times, func(i, j int) bool {
		return times[i].Before(times[j])
	})

	latest := times[len(times)-1]
	name := latest.Format(FileFormat)

	filepath := path.Join(dirpath, name)
	return &filepath, nil
}

func parseFileNames(files []os.FileInfo) []time.Time {
	times := make([]time.Time, len(files))
	for i, file := range files {
		name := file.Name()
		t, err := time.Parse(FileFormat, name)
		if err != nil {
			log.Println("An error ocurred reading through", name, err)
			continue
		}
		times[i] = t
	}
	return times
}

func read(filepath string) (*Durable, error) {
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var d Durable

	err = json.Unmarshal(b, &d)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

func (d *Durable) Write() error {
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}

	now := time.Now()
	name := now.Format(FileFormat)
	path := path.Join(d.dirpath, name)

	return ioutil.WriteFile(path, b, 644)
}
