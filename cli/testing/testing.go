// Testing package provides helpers to facilitate rig testing.
// See additional documentation: https://gist.github.com/grayside/ffeb68fa342cecf1ec158c011cbd2ea3
package testing

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// ExecMockSet provides a set of unique mocks. The key matches the remote execution script.
type ExecMockSet map[string]string

// ExecMockCollection is a map of ExecMockSets. The key is a category of mocks, such as
// "success" or "notfound".
type ExecMockCollection map[string]ExecMockSet

var mock ExecMockCollection

// SetMockValues provides an easy setter that allows the TestMain implementation
// of individual test files to preload the potential values to use.
func SetMockByType(namespace string, mockSet ExecMockSet) {
	if mock == nil {
		mock = make(ExecMockCollection)
	}
	mock[namespace] = mockSet
}

// TestMain is a special function that takes over handling the behavior of the the test runner `go test`
// generates to execute your code. I do not know if you can have one per file or one for a project's entire
// collection of tests.
//
// In the example below, we rely on an environment variable: `GO_TEST_MODE` to determine whether the
// testing process will behave normally (running all test and handling the result, done by default) or
// will behavior in a special manner because we have tailored the way an exec.Command() will execute
// to flow through this logic instead of what was originally intended.
//
// You may be wondering, why would we go to such an elaborate length to mock the result of the a shell
// execution? Well, if we directly interpolated the mocked value for the command, the resulting object
// would be a string, and not the expected structure the code might be looking for as a result of executing
// a remote command.
//
// To use this function, implement TestMain in your own class, then call:
//
// testing.MainTestProcess(m)
func MainTestProcess(m *testing.M) {
	switch os.Getenv("GO_TEST_MODE") {
	case "":
		// Normal test mode.
		os.Exit(m.Run())

	case "echo":
		// Outputs the arguments passed to the test runner.
		// This will be the command that would have executed under normal runtime.
		// This mode can be used to test that we can predict programmatically assembled command that would be executed.
		fmt.Println(strings.Join(os.Args[1:], " "))

	case "succeed":
		os.Exit(0)

	case "fail":
		os.Exit(42)

	case "mock":
		if mock != nil {
			// Used the command that would be executed under normal runtime as the key to our mock value map and outputs the value.
			// I am still researching how to adjust this overall pattern to centralize the code as test helpers but allow individual
			// test files to supply their own mock.
			fmt.Printf("%s", mock["success"][strings.Join(os.Args[1:], " ")])
		}
	}
}

// MockExecCommand uses fakeExecCommand to transform the intended remote execution
// into something controlled by the test runner, then adds an environment variable to
// the command so TestMain routes it to the command "mock" functionality.
func MockExecCommand(command string, args ...string) *exec.Cmd {
	cmd := fakeExecCommand(command, args...)
	cmd.Env = append(cmd.Env, "GO_TEST_MODE=mock")
	return cmd
}

// EchoExecCommand uses fakeExecCommand to transform the intended remote execution
// into something controlled by the test runner, then adds an environment variable to
// the command so TestMain routes it to the command "echo" functionality.
func EchoExecCommand(command string, args ...string) *exec.Cmd {
	cmd := fakeExecCommand(command, args...)
	cmd.Env = append(cmd.Env, "GO_TEST_MODE=echo")
	return cmd
}

// SucceedExecCommand uses fakeExecCommand to transform the intended remote execution
// into something controlled by the test runner, then adds an environment variable to
// the command so TestMain routes it to the command "success" functionality.
func SuccessExecCommand(command string, args ...string) *exec.Cmd {
	cmd := fakeExecCommand(command, args...)
	cmd.Env = append(cmd.Env, "GO_TEST_MODE=success")
	return cmd
}

// FailExecCommand uses fakeExecCommand to transform the intended remote execution
// into something controlled by the test runner, then adds an environment variable to
// the command so TestMain routes it to the command "fail" functionality.
func FailExecCommand(command string, args ...string) *exec.Cmd {
	cmd := fakeExecCommand(command, args...)
	cmd.Env = append(cmd.Env, "GO_TEST_MODE=fail")
	return cmd
}

// fakeExecCommand creates a new reference to an exec.Cmd object which has been transformed
// to use the supplied parameters as arguments to be submitted to our test runner binary.
// It should never be used directly.
func fakeExecCommand(command string, args ...string) *exec.Cmd {
	testArgs := []string{command}
	testArgs = append(testArgs, args...)
	cmd := exec.Command(os.Args[0], testArgs...)
	cmd.Env = []string{}

	return cmd
}
