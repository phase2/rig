package util

import (
	"log"
	"os"
	"github.com/fatih/color"
)

var logger *RigLogger

type RigLogger struct {
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
}

func Logger() *RigLogger {
	if logger == nil {
		logger = &RigLogger{
			Info:    log.New(os.Stdout, color.BlueString("[INFO] "), 0),
			Warning: log.New(os.Stdout, color.YellowString("[WARN] "), 0),
			Error:   log.New(os.Stderr, color.RedString("[ERROR] "), 0),
		}
	}

	return logger
}



