package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kardianos/osext"
)

// GetExecutableDir returns the directory of this binary
func GetExecutableDir() (string, error) {
	return osext.ExecutableFolder()
}

// AbsJoin joins the two path segments, ensuring they form an absolute path.
func AbsJoin(baseDir, suffixPath string) (string, error) {
	if len(baseDir) == 0 {
		baseDir = string(filepath.Separator)
	}
	absoluteBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("Unrecognized working directory: %s: %s", baseDir, err.Error())
	}

	return filepath.Join(absoluteBaseDir, suffixPath), nil
}

// FileExists reports whether a file exists.
func FileExists(pathToFile, workingDir string) bool {
	absoluteFilePath, err := AbsJoin(workingDir, pathToFile)
	if err == nil {
		_, statErr := os.Stat(absoluteFilePath)
		return statErr == nil
	}

	return false
}

// RemoveFile removes the designated file relative to the Working Directory.
func RemoveFile(pathToFile, workingDir string) error {
	absoluteFilePath, err := AbsJoin(workingDir, pathToFile)
	if err != nil {
		return err
	}

	return os.Remove(absoluteFilePath)
}

// RemoveFileGlob removes all files under the working directory that match the glob.
// This recursively traverses all sub-directories. If a logger is passed the action
// will be verbosely logged, otherwise pass nil to skip all output.
func RemoveFileGlob(glob, targetDirectory string, logger *RigLogger) error {
	return filepath.Walk(targetDirectory, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			globPath := filepath.Join(path, glob)
			if files, globErr := filepath.Glob(globPath); globErr == nil {
				for _, file := range files {
					if logger != nil {
						logger.Verbose("Removing file '%s'...", file)
					}
					if removeErr := RemoveFile(file, ""); removeErr != nil {
						logger.Verbose("Remove error '%s'", removeErr)
						return removeErr
					}
				}
			} else {
				return globErr
			}
		}

		return nil
	})
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

// LoadFile loads a file into an array, without the newlines
func LoadFile(filename string) []string {
	lines := make([]string, 0, 1000)
	f, err := os.Open(filename)
	if err != nil {
		return lines
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		switch ln, err := r.ReadString('\n'); err {
		case nil:
			ln = strings.Replace(ln, "\r", "", -1)
			ln = strings.Replace(ln, "\n", "", -1)
			lines = append(lines, ln)
		case io.EOF:
			return lines
		default:
			fmt.Println(err)
		}
	}
}
