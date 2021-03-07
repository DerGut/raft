package raft

import (
	"log"
	"os"
)

const flagSet = log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile

var (
	// Debug level logger logs to stdout
	Debug *log.Logger
	// Info level logger logs to stdout
	Info *log.Logger
	// Error level logger logs to stderr
	Error *log.Logger
)

func init() {
	Debug = log.New(os.Stdout, "INFO:", flagSet)
	Info = log.New(os.Stdout, "INFO:", flagSet)
	Error = log.New(os.Stderr, "ERROR:", flagSet)
}
