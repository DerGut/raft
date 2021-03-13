package state

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"time"
)

const (
	filesToKeep     = 2
	cleanUpInterval = 10 * time.Second
)

type Durable struct {
	dirpath string

	CurrentTerm Term    `json`
	VotedFor    *string `json`
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

	return writeToTimestampedFile(d.dirpath, b)
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
	if err := deleteOldFiles(d.dirpath); err != nil {
		log.Println("Error cleaning files", err)
	}
}
