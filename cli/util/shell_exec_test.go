package util_test

import (
	"testing"

	rigtest "github.com/phase2/rig/cli/testing"
	"github.com/phase2/rig/cli/util"
)

// TestPassThruCommand confirms we receive the exit code.
// For more thoroughly commented exec wrangling details see docker_test.go::TestGetRawCurrentDockerVersion.
func TestPassthruCommand(t *testing.T) {
	actual := util.PassthruCommand(rigtest.SuccessExecCommand("ls"))
	rigtest.Equals(t, 0, actual)

	actual = util.PassthruCommand(rigtest.FailExecCommand("ls"))
	rigtest.Equals(t, 42, actual)
}
