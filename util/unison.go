package util

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/go-version"
)

// GetUnisonMinorVersion will return the local Unison version to try to load a compatible unison image
// This function discovers a semver like 2.48.4 and return 2.48
func GetUnisonMinorVersion() string {
	output, _ := Command("unison", "-version").Output()
	re := regexp.MustCompile(`unison version (\d+\.\d+\.\d+)`)
	rawVersion := re.FindAllStringSubmatch(string(output), -1)[0][1]
	v := version.Must(version.NewVersion(rawVersion))
	segments := v.Segments()
	return fmt.Sprintf("%d.%d", segments[0], segments[1])
}
