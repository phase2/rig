package util

// IndexOfString is a general utility function that can find the index of a value
// present in a slice. The type of the slice does not matter. The second value
// is true if the item is found, which means this can be used either as a index
// position check or as a value existence check.
func IndexOfString(slice []string, value string) (int, bool) {
	for index, elem := range slice {
		if elem == value {
			return index, true
		}
	}

	return 0, false
}
