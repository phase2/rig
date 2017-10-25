package util

import (
    "testing"

    rigtest "github.com/phase2/rig/cli/testing"
)

// mock provides mock values to use as lookup responses to functions we will execute in our production code.
// The idea is to use the command as a lookup key to the result it might generate.
// Currently it only supports a single value, in the future this may be split into multiple maps for different,
// generic classes of success and failure. We cannot use multiple values for entries in this map because each response
// is expected to be a string that an executed command would return to Stdout.
var mockSet = rigtest.ExecMockSet{
	"docker --version": "Docker version 17.09.0-ce, build afdb6d4",
  "docker-machine ssh gastropod docker version --format {{.Server.APIVersion}}": "1.30",
  "docker version --format {{.Client.APIVersion}}": "1.30",
  "docker-machine ssh gastropod docker version --format {{.Server.MinAPIVersion}}": "1.12",
  "docker inspect --format {{.Created}} outrigger/dust": "2017-09-18T21:43:00.565978065Z",
}

func TestMain(m *testing.M) {
  rigtest.SetMockByType("success", mockSet)
  rigtest.MainTestProcess(m)
}

// TestGetRawCurrentDockerVersion confirms successful Docker version extraction.
func TestGetRawCurrentDockerVersion(t *testing.T) {
  // In case some other functionality has swapped out this value, we will store
  // it explicitly rather than assume it is exec.Command.
  stashCommand := execCommand
  // Re-define execCommand so our runtime code executes using the mocking functionality.
  // I thought execCommand would be a private variable in file scope, apparently sharing the package
  // is enough to access and manipulate it. Or perhaps test functions have special scope rules?
  execCommand = rigtest.MockExecCommand
  // Put back the original behavior after we are done with this test function.
  defer func(){ execCommand = stashCommand }()
  // Run the code under test.
  out := GetRawCurrentDockerVersion()

  // Implement our assertion.
  expected := "17.09.0-ce"
  if out != expected {
	  t.Errorf("GetRawCurrentDockerVersion: Expected %q, Actual %q", expected, out)
  }
}

// TestGetCurrentDockerVersion confirms successful processing of Docker version into version object.
// For more thoroughly commented exec wrangling details see TestGetRawCurrentDockerVersion.
func TestGetCurrentDockerVersion(t *testing.T) {
  stashCommand := execCommand
  execCommand = rigtest.MockExecCommand
  defer func(){ execCommand = stashCommand }()
  version, err := GetDockerServerApiVersion("gastropod")

  if err != nil {
      t.Errorf("GetDockerServerApiVersion: %v", err)
  }

  expected := "1.30.0"
  if version.String() != expected {
    t.Errorf("GetDockerServerApiVersion: Expected %q, Actual %q", expected, version)
  }
}

// TestGetDockerServerApiVersion confirms successful Docker client version extraction.
// For more thoroughly commented exec wrangling details see TestGetRawCurrentDockerVersion.
func TestGetDockerClientApiVersion(t *testing.T) {
  stashCommand := execCommand
  execCommand = rigtest.MockExecCommand
  defer func(){ execCommand = stashCommand }()
  version := GetDockerClientApiVersion()

  expected := "1.30.0"
  if version.String() != expected {
    t.Errorf("GetDockerClientApiVersion: Expected %q, Actual %q", expected, version)
  }
}

// TestGetDockerServerApiVersion confirms successful Docker server version extraction.
// For more thoroughly commented exec wrangling details see TestGetRawCurrentDockerVersion.
func TestGetDockerServerApiVersion(t *testing.T) {
  stashCommand := execCommand
  execCommand = rigtest.MockExecCommand
  defer func(){ execCommand = stashCommand }()
  version, err := GetDockerServerApiVersion("gastropod")

  if err != nil {
      t.Errorf("GetDockerServerApiVersion: %v", err)
  }

  expected := "1.30.0"
  if version.String() != expected {
	  t.Errorf("GetDockerServerApiVersion: Expected %q, Actual %q", expected, version)
  }
}

// TestGetDockerServerMinApiVersion confirms successful Docker minimum API compatibility version.
// For more thoroughly commented exec wrangling details see TestGetRawCurrentDockerVersion.
func TestGetDockerServerMinApiVersion(t *testing.T) {
  stashCommand := execCommand
  execCommand = rigtest.MockExecCommand
  defer func(){ execCommand = stashCommand }()
  version, err := GetDockerServerMinApiVersion("gastropod")

  if err != nil {
      t.Errorf("GetDockerServerMinApiVersion: %v", err)
  }

  expected := "1.12.0"
  if version.String() != expected {
	  t.Errorf("GetDockerServerMinApiVersion: Expected %q, Actual %q", expected, version)
  }
}

// TestImageOlderThan confirms image age evaluation.
// For more thoroughly commented exec wrangling details see TestGetRawCurrentDockerVersion.
// @TODO identify how to mock the current time so we can test this more completely.
func TestImageOlderThan(t *testing.T) {
  stashCommand := execCommand
  execCommand = rigtest.MockExecCommand
  defer func(){ execCommand = stashCommand }()
  older, _, err := ImageOlderThan("outrigger/dust", 86400*30)

  if err != nil {
      t.Errorf("ImageOlderThan: %v", err)
  }

  if !older {
    t.Errorf("ImageOlderThan: Image is older than 30 days ago but reporting as newer.")
  }
}
