package util

import (
	"testing"
)

func TestLoggerInit(t *testing.T) {
	LoggerInit(false)
	logger := Logger()
	if logger.IsVerbose {
		t.Error("Logger initialized in non-Verbose mode.")
	}

	LoggerInit(true)
	logger = Logger()
	if !logger.IsVerbose {
		t.Error("Logger initialized in Verbose mode.")
	}
}
