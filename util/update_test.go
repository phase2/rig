package util

import (
	"fmt"
	"testing"
)

func TestUpgradeCheck(t *testing.T) {
	Logger().SetVerbose(true)
	// Local dev uses version "master"
	if msg := CheckForRigUpdate("master"); msg == "" {
		t.Error("well that didn't work.")
	} else {
		fmt.Println(msg)
	}
	// Test no update available
	if curVersion, err := currentRigReleaseTag(); err != nil {
		t.Error(err)
	} else if msg := CheckForRigUpdate(curVersion); msg != "" {
		t.Error(msg)
	}
}
