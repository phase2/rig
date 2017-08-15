package util

import (
	"os/exec"
	"strings"
	"time"
)

func ImageOlderThan(image string, elapsed_seconds float64) (bool, float64, error) {
	output, err := exec.Command("docker", "inspect", "--format", "{{.Created}}", image).Output()
	if err != nil {
		return false, 0, err
	}

	datestring := strings.TrimSpace(string(output))
	datetime, err := time.Parse(time.RFC3339, datestring)
	if err != nil {
		return false, 0, err
	}

	seconds := time.Since(datetime).Seconds()
	return seconds > elapsed_seconds, seconds, nil
}
