package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kardianos/osext"
)

// GetExecutableDir returns the directory of this binary
func GetExecutableDir() (string, error) {
	return osext.ExecutableFolder()
}

// AbsJoin joins the two path segments, ensuring they form an absolute path.
func AbsJoin(baseDir string, suffixPath string) (string, error) {
	absoluteBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("Unrecognized working directory: %s: %s", baseDir, err.Error())
	}

	return filepath.Join(absoluteBaseDir, suffixPath), nil
}

// RemoveFile removes the designated file relative to the Working Directory.
func RemoveFile(pathToFile string, workingDir string) error {
	absoluteFilePath, err := AbsJoin(workingDir, pathToFile)
	if err != nil {
		return err
	}

	return os.Remove(absoluteFilePath)
}

// TouchFile creates an empty file, usually for temporary use.
// @see https://stackoverflow.com/questions/35558787/create-an-empty-text-file/35558965
func TouchFile(pathToFile string, workingDir string) error {
	absoluteFilePath, err := AbsJoin(workingDir, pathToFile)
	if err != nil {
		return err
	}

	// If the file already exists there will be no error.
	f, err := os.OpenFile(absoluteFilePath, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("Could not touch file: %s: %s", absoluteFilePath, err.Error())
	}
	// Not checking for an error here because we are not very currently concerned
	// with file descriptor leaks
	f.Close()
	return nil
}
