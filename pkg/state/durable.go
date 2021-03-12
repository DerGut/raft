package state

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	fileFormat      = time.RFC3339
	filesToKeep     = 2
	cleanUpInterval = 10 * time.Second
)

type Durable struct {
	dirpath string

	CurrentTerm Term    `json:`
	VotedFor    *string `json:`
	Log         `json`
}

var errNoStateFile = errors.New("no state file")

func NewDurable(dirpath string) (*Durable, error) {
	file, err := latestWrite(dirpath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println(dirpath, "does not exist, creating")
			if err = os.MkdirAll(dirpath, os.ModePerm); err != nil {
				return nil, err
			}
			return &Durable{dirpath: dirpath}, nil
		} else if errors.Is(err, errNoStateFile) {
			log.Println("No state file present in", dirpath, "creating new Durable")
			return &Durable{dirpath: dirpath}, nil
		}
		return nil, err
	}

	d, err := read(*file)
	if err != nil {
		return nil, err
	}
	d.dirpath = dirpath
	return d, nil
}

func latestWrite(dirpath string) (file *string, err error) {
	files, err := orderedFiles(dirpath)
	if err != nil {
		return nil, err
	}

	return &files[len(files)-1], nil
}

func orderedFiles(dirpath string) ([]string, error) {
	files, err := ioutil.ReadDir(dirpath)
	if err != nil {
		return nil, err
	}

	times := parseFileNames(files)
	if len(times) == 0 {
		return nil, errNoStateFile
	}

	sort.Slice(times, func(i, j int) bool {
		return times[i].Before(times[j])
	})

	f := make([]string, len(times))
	for i, time := range times {
		name := time.Format(fileFormat)
		f[i] = filepath.Join(dirpath, name)
	}

	return f, nil
}

func parseFileNames(files []os.FileInfo) []time.Time {
	times := make([]time.Time, len(files))
	for i, file := range files {
		name := file.Name()
		t, err := time.Parse(fileFormat, name)
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
	name := now.Format(fileFormat)
	path := filepath.Join(d.dirpath, name)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(b)
	if err != nil {
		return err
	}

	return f.Sync()
}

func (d *Durable) Clean(done chan struct{}) {
	t := time.NewTicker(cleanUpInterval)
	for {
		select {
		case <-t.C:
			d.clean()
		case <-done:
			t.Stop()
			return
		}
	}
}

func (d *Durable) clean() {
	files, err := orderedFiles(d.dirpath)
	if err != nil {
		log.Println("Error cleaning files", err)
	}
	if len(files) <= filesToKeep {
		return
	}

	toClean := files[:len(files)-filesToKeep]
	for _, file := range toClean {
		err = os.Remove(file)
		if err != nil {
			log.Println("Failed to remove old state", err)
		}
	}
}
