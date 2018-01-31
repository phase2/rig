package util

import (
	"strings"
)

// IndexOfString is a general utility function that can find the index of a value
// present in a string slice. The second value is true if the item is found.
func IndexOfString(slice []string, search string) (int, bool) {
	for index, elem := range slice {
		if elem == search {
			return index, true
		}
	}

	return 0, false
}

// IndexOfSubstring is a variation on IndexOfString which checks to see if a
// given slice value matches our search string, or if that search string is
// a substring of the element. The second value is true if the item is found.
func IndexOfSubstring(slice []string, search string) (int, bool) {
	for index, elem := range slice {
		if strings.Contains(elem, search) {
			return index, true
		}
	}

	return 0, false
}
