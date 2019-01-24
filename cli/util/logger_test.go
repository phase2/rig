package util_test

import (
	"testing"

	rigtest "github.com/phase2/rig/cli/testing"
	"github.com/phase2/rig/cli/util"
)

func TestLoggerInit(t *testing.T) {
	util.LoggerInit(false)
	logger := util.Logger()
	rigtest.Assert(t, !logger.IsVerbose, "Logger initialized in Verbose mode.")

	util.LoggerInit(true)
	logger = util.Logger()
	rigtest.Assert(t, logger.IsVerbose, "Logger initialized in non-Verbose mode.")
}
