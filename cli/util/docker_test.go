package util_test

import (
	"testing"

	rigtest "github.com/phase2/rig/cli/testing"
	"github.com/phase2/rig/cli/util"
)

// mock provides mock values to use as lookup responses to functions we will execute in our production code.
// The idea is to use the command as a lookup key to the result it might generate.
// Currently it only supports a single value, in the future this may be split into multiple maps for different,
// generic classes of success and failure. We cannot use multiple values for entries in this map because each response
// is expected to be a string that an executed command would return to Stdout.
var mockSet = rigtest.ExecMockSet{
	"docker --version": "Docker version 17.09.0-ce, build afdb6d4",
	"docker-machine ssh gastropod docker version --format {{.Server.APIVersion}}":    "1.30",
	"docker version --format {{.Client.APIVersion}}":                                 "1.30",
	"docker-machine ssh gastropod docker version --format {{.Server.MinAPIVersion}}": "1.12",
	"docker inspect --format {{.Created}} outrigger/dust":                            "2017-09-18T21:43:00.565978065Z",
}

func init() {
	rigtest.SetMockByType("success", mockSet)
}

// TestGetRawCurrentDockerVersion confirms successful Docker version extraction.
func TestGetRawCurrentDockerVersion(t *testing.T) {
	// In case some other functionality has swapped out this value, we will store
	// it explicitly rather than assume it is exec.Command.
	stashCommand := util.ExecCommand
	// Re-define util.ExecCommand so our runtime code executes using the mocking functionality.
	// I thought util.ExecCommand would be a private variable in file scope, apparently sharing the package
	// is enough to access and manipulate it. Or perhaps test functions have special scope rules?
	util.ExecCommand = rigtest.MockExecCommand
	// Put back the original behavior after we are done with this test function.
	defer func() { util.ExecCommand = stashCommand }()
	// Run the code under test.
	actual := util.GetRawCurrentDockerVersion()
	rigtest.Equals(t, "17.09.0-ce", actual)
}

// TestGetCurrentDockerVersion confirms successful processing of Docker version into version object.
// For more thoroughly commented exec wrangling details see TestGetRawCurrentDockerVersion.
func TestGetCurrentDockerVersion(t *testing.T) {
	stashCommand := util.ExecCommand
	util.ExecCommand = rigtest.MockExecCommand
	defer func() { util.ExecCommand = stashCommand }()
	actual, err := util.GetDockerServerApiVersion("gastropod")
	rigtest.Ok(t, err)
	rigtest.Equals(t, "1.30.0", actual.String())
}

// TestGetDockerServerApiVersion confirms successful Docker client version extraction.
// For more thoroughly commented exec wrangling details see TestGetRawCurrentDockerVersion.
func TestGetDockerClientApiVersion(t *testing.T) {
	stashCommand := util.ExecCommand
	util.ExecCommand = rigtest.MockExecCommand
	defer func() { util.ExecCommand = stashCommand }()
	actual := util.GetDockerClientApiVersion()
	rigtest.Equals(t, "1.30.0", actual.String())
}

// TestGetDockerServerApiVersion confirms successful Docker server version extraction.
// For more thoroughly commented exec wrangling details see TestGetRawCurrentDockerVersion.
func TestGetDockerServerApiVersion(t *testing.T) {
	stashCommand := util.ExecCommand
	util.ExecCommand = rigtest.MockExecCommand
	defer func() { util.ExecCommand = stashCommand }()
	actual, err := util.GetDockerServerApiVersion("gastropod")
	rigtest.Ok(t, err)
	rigtest.Equals(t, "1.30.0", actual.String())
}

// TestGetDockerServerMinApiVersion confirms successful Docker minimum API compatibility version.
// For more thoroughly commented exec wrangling details see TestGetRawCurrentDockerVersion.
func TestGetDockerServerMinApiVersion(t *testing.T) {
	stashCommand := util.ExecCommand
	util.ExecCommand = rigtest.MockExecCommand
	defer func() { util.ExecCommand = stashCommand }()
	actual, err := util.GetDockerServerMinApiVersion("gastropod")
	rigtest.Ok(t, err)
	rigtest.Equals(t, "1.12.0", actual.String())
}

// TestImageOlderThan confirms image age evaluation.
// For more thoroughly commented exec wrangling details see TestGetRawCurrentDockerVersion.
// @TODO identify how to mock the current time so we can test this more completely.
func TestImageOlderThan(t *testing.T) {
	stashCommand := util.ExecCommand
	util.ExecCommand = rigtest.MockExecCommand
	defer func() { util.ExecCommand = stashCommand }()
	older, _, err := util.ImageOlderThan("outrigger/dust", 86400*30)
	rigtest.Ok(t, err)
	rigtest.Assert(t, older, "Image is older than 30 days ago but reporting as newer.", "howdy")
}
