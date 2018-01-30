package util

// IndexOfString is a general utility function that can find the index of a value
// present in a string slice. The second value is true if the item is found.
func IndexOfString(slice []string, value string) (int, bool) {
	for index, elem := range slice {
		if elem == value {
			return index, true
		}
	}

	return 0, false
}
