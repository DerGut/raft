package app

import (
	"errors"
	"fmt"
	"strings"
)

type URIs []string

func (u *URIs) String() string {
	return fmt.Sprint(*u)
}

func (u *URIs) Set(value string) error {
	if len(*u) > 0 {
		return errors.New("URIs flag already set")
	}
	for _, v := range strings.Split(value, ",") {
		*u = append(*u, v)
	}
	return nil
}
