package util

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/fatih/color"
)

var logger *RigLogger

// RigLogger is the global logger object
type RigLogger struct {
	Info      *log.Logger
	Warning   *log.Logger
	Error     *log.Logger
	Verbose   *log.Logger
	IsVerbose bool
}

// LoggerInit initializes the global logger
func LoggerInit(verbose bool) {
	var verboseWriter = ioutil.Discard
	if verbose {
		verboseWriter = os.Stdout
	}
	logger = &RigLogger{
		Info:      log.New(os.Stdout, color.BlueString("[INFO] "), 0),
		Warning:   log.New(os.Stdout, color.YellowString("[WARN] "), 0),
		Error:     log.New(os.Stderr, color.RedString("[ERROR] "), 0),
		Verbose:   log.New(verboseWriter, "[VERBOSE] ", 0),
		IsVerbose: verbose,
	}
}

// Logger returns the instance of the global logger
func Logger() *RigLogger {
	if logger == nil {
		LoggerInit(false)
	}

	return logger
}
