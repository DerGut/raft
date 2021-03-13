package app

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
)

type KeyValueStore struct {
	Entries map[string]string `json`
}

func (s *KeyValueStore) Commit(cmd string) (result string) {
	log.Println("Committed:", cmd)
	result, err := s.execute(cmd)
	if err != nil {
		log.Println("Failed to run cmd", cmd, err)
	}
	log.Println("Now:", s.Entries)
	return
}

func (s *KeyValueStore) execute(cmd string) (string, error) {
	cmd = strings.TrimSpace(cmd)
	cmd = strings.ToLower(cmd)
	split := strings.Split(cmd, " ")
	switch split[0] {
	case "get":
		if len(split) > 2 {
			return "", errors.New("get cmd has too many entries" + cmd)
		}
		return s.Entries[split[1]], nil
	case "set":
		if len(split) < 3 {
			return "", errors.New("set cmd has too few entries" + cmd)
		}
		s.Entries[split[1]] = split[2]
		return "", nil
	default:
		return "", errors.New("op does not exist" + split[0])
	}
}

func (s *KeyValueStore) Snapshot() []byte {
	b, err := json.Marshal(s.Entries)
	if err != nil {
		log.Println("Could not marshal key value store entries", err)
		return []byte{}
	}

	return b
}

func (s *KeyValueStore) Install(data []byte) {
	var m map[string]string
	err := json.Unmarshal(data, &m)
	if err != nil {
		log.Println("Failed to install snapshot", err)
	}
	s.Entries = m
}
