package app

import (
	"hash/fnv"
	"math/rand"
	"time"
)

func SeedRand(address string) error {
	hashAlg := fnv.New64()
	b := []byte(address)
	if _, err := hashAlg.Write(b); err != nil {
		return err
	}
	if _, err := hashAlg.Write([]byte(time.Now().String())); err != nil {
		return err
	}
	hashSum := hashAlg.Sum64()

	rand.Seed(int64(hashSum))
	return nil
}
