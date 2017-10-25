package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kardianos/osext"
)

// Get the directory of this binary
func GetExecutableDir() (string, error) {
	return osext.ExecutableFolder()
}

// AbsJoin joins the two path segments, ensuring they form an absolute path.
func AbsJoin(baseDir string, suffixPath string) (string, error) {
	if absoluteBaseDir, err := filepath.Abs(baseDir); err != nil {
		return "", fmt.Errorf("Unrecognized working directory: %s: %s", baseDir, err.Error())
	} else {
		return filepath.Join(absoluteBaseDir, suffixPath), nil
	}
}

// RemoveFile removes the designated file relative to the Working Directory.
func RemoveFile(pathToFile string, workingDir string) error {
	if absoluteFilePath, err := AbsJoin(workingDir, pathToFile); err != nil {
		return err
	} else {
		return os.Remove(absoluteFilePath)
	}

	return nil
}

// TouchFile creates an empty file, usually for temporary use.
// @see https://stackoverflow.com/questions/35558787/create-an-empty-text-file/35558965
func TouchFile(pathToFile string, workingDir string) error {
	if absoluteFilePath, err := AbsJoin(workingDir, pathToFile); err != nil {
		return err
	} else {
		// If the file already exists there will be no error.
		if f, err := os.OpenFile(absoluteFilePath, os.O_RDONLY|os.O_CREATE, 0666); err != nil {
			return fmt.Errorf("Could not touch file: %s: %s", absoluteFilePath, err.Error())
		} else {
			// Not checking for an error here because we are not very currently concerned
			// with file descriptor leaks
			f.Close()
		}
	}

	return nil
}
