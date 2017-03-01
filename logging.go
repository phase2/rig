package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/fatih/color"
)

type RigLogger struct {
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
	Verbose *log.Logger
}

var out RigLogger
var verboseWriter io.Writer
var verboseMode bool

func LoggerCreate(
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer,
	verboseHandle io.Writer) {

	out = RigLogger{
		Info:    log.New(infoHandle, color.BlueString("[INFO] "), 0),
		Warning: log.New(warningHandle, color.YellowString("[WARN] "), 0),
		Error:   log.New(errorHandle, color.RedString("[ERROR] "), 0),
		Verbose: log.New(verboseHandle, "[VERBOSE] ", 0),
	}
}

func LoggerInit(verbose bool) {
	verboseWriter = ioutil.Discard
	if verbose {
		verboseWriter = os.Stdout
		verboseMode = true
	}
	LoggerCreate(os.Stdout, os.Stdout, os.Stderr, verboseWriter)
}
