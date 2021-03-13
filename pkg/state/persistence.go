package state

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const fileFormat = time.StampMicro

func writeToTimestampedFile(dirpath string, data []byte) error {
	name := timestampedFileName()
	f, err := os.Create(filepath.Join(dirpath, name))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	return f.Sync()
}

func timestampedFileName() string {
	now := time.Now()
	return now.Format(fileFormat)
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
			log.Println("An error ocurred parsing", name, err)
			continue
		}
		times[i] = t
	}
	return times
}

func deleteOldFiles(dirpath string) error {
	files, err := orderedFiles(dirpath)
	if err != nil {
		return err
	}
	if len(files) <= filesToKeep {
		return nil
	}

	toClean := files[:len(files)-filesToKeep]
	for _, file := range toClean {
		err = os.Remove(file)
		if err != nil {
			return err
		}
	}

	return nil
}
