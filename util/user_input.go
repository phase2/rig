package util

import (
	"fmt"
)

// AskYesNo asks the user a yes/no question
// Return true if they answered yes, false otherwise
func AskYesNo(question string) bool {

	fmt.Printf("%s? [y/N]: ", question)

	var response string
	fmt.Scanln(&response)

	yesResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	for _, elem := range yesResponses {
		if elem == response {
			return true
		}
	}
	return false
}
