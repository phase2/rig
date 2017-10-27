package util_test

import (
	"testing"

	rigtest "github.com/phase2/rig/cli/testing"
)

// Controls the test execution fo the util sub-package.
// Note that if tests were to be run for the entire package cross-package
// duplication of this function would cause it to explode.
func TestMain(m *testing.M) {
	rigtest.MainTestProcess(m)
}
