package util

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/fatih/color"
)

// PrintDebugHelp provides expanded troubleshooting help content for an error.
// It is primarily called by command.go:Failure().
// @todo consider switching this to a template.
func PrintDebugHelp(message, errorName string, exitCode int) {
	header := color.New(color.FgYellow).Add(color.Underline).PrintlnFunc()
	red := color.New(color.FgRed).PrintlnFunc()
	code := color.New(color.BgHiBlack).PrintfFunc()

	header(StringPad(fmt.Sprintf("Error [%s]", errorName), " ", 80))
	fmt.Println()
	red(color.RedString(message))

	var codeMessage string
	switch exitCode {
	case 12:
		codeMessage = "environmental"
	case 13:
		codeMessage = "external/upstream command"
	case 418:
		codeMessage = "rig developer test command"
	default:
		codeMessage = "general"
	}
	fmt.Println()
	fmt.Printf("This is a %s error.\n", codeMessage)
	fmt.Println()

	header(StringPad("Debugging Help", " ", 80))
	fmt.Println()
	if !Logger().IsVerbose {
		fmt.Println("Run again in verbose mode:")
		fmt.Println()
		line := fmt.Sprintf("%s --verbose %s", os.Args[0], strings.Join(os.Args[1:], " "))
		code("\t %s", StringPad("", " ", len(line)+1))
		fmt.Println()
		code("\t %s", StringPad(line, " ", len(line)+1))
		fmt.Println()
		code("\t %s", StringPad("", " ", len(line)+1))
		fmt.Println()
		fmt.Println()
	}
	fmt.Println("Ask the doctor for a general health check:")
	fmt.Println()
	line := "rig doctor"
	code("\t %s", StringPad("", " ", len(line)+1))
	fmt.Println()
	code("\t %s", StringPad(line, " ", len(line)+1))
	fmt.Println()
	code("\t %s", StringPad("", " ", len(line)+1))
	fmt.Println()
	fmt.Println()

	header(StringPad("Get Support", " ", 80))
	fmt.Println()
	fmt.Printf("To search for related issues or documentation use the error ID '%s'.\n", errorName)
	fmt.Println()
	fmt.Println("\tDocs:\t\thttp://docs.outrigger.sh")
	fmt.Println("\tIssues:\t\thttps://github.com/phase2/rig/issues")
	fmt.Println("\tChat:\t\thttp://slack.outrigger.sh/")
	fmt.Println()

	header(StringPad("Your Environment Information", " ", 80))
	fmt.Println()
	// Verbose output is distracting in this help output.
	Logger().SetVerbose(false)
	fmt.Println("\tOperating System:\t\t", runtime.GOOS)
	fmt.Println("\tdocker version:\t\t\t", GetCurrentDockerVersion())
	fmt.Println("\tdocker client API version:\t", GetDockerClientAPIVersion())
	if version, err := GetDockerServerAPIVersion(); err == nil {
		fmt.Println("\tdocker server API version:\t", version)
	} else {
		fmt.Println("\tdocker server API version:\t", err.Error())
	}
	fmt.Println(color.CyanString("\nPlease include the 'Error' and 'Your Environment Information' sections in bug reports."))
	fmt.Println()
	fmt.Println("To disable the extended troubleshooting output, run with --power-user or RIG_POWER_USER_MODE=1")
	fmt.Println()
}

// StringPad takes your string and returns it with the pad value repeatedly
// appended until it is the intended length. Note that if the delta between
// the initial string length and the intended size is not evenly divisible by
// the pad length, your final string could be slightly short -- partial padding
// is not applied. For guaranteed results, use a pad string of length 1.
func StringPad(s string, pad string, size int) string {
	length := len(s)
	if length < size {
		var buffer bytes.Buffer
		padLength := len(pad)
		delta := size - length
		iterations := delta / padLength

		buffer.WriteString(s)
		for i := 0; i <= iterations; i += padLength {
			buffer.WriteString(pad)
		}

		return buffer.String()
	}

	return s
}
