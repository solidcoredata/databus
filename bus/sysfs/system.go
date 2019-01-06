package sysfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"solidcoredata.org/src/databus/bus"
)

/*
   /src (input)
   /version (output)
   /run (runner)
       /runner-name.com/sql
       /runner-name.com/ui
*/

// NewSystem creates new FS components from the givent fs root directory.
// The root directory can be obtained from RootFromWD.
func NewSystem(root string) bus.System {
	sys := bus.System{
		Input:  NewInput(root),
		Output: NewOutput(root),
		Runner: NewRunner(root),
	}
	return sys
}

const (
	configFilename = "scd.jsonnet"
)

// RootFromWD starts in the current working directory and looks for the root
// of the bus directory. If not found it returns an error.
// If projectRoot is empty, the current working directory is used as the start.
func RootFromWD(projectRoot string) (string, error) {
	if len(projectRoot) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		projectRoot = wd
	}
	current := filepath.Clean(projectRoot)
	for i := 0; i < 1000; i++ { // Prevent looping forever in case of logic bug.
		if len(current) == 0 {
			break
		}
		try := filepath.Join(current, configFilename)
		_, err := os.Stat(try)
		if err == nil {
			return current, nil
		}
		current = removeSegment(current)
	}
	return "", fmt.Errorf("unable to find project root starting in %q", projectRoot)
}

var filepathSeparator = string(filepath.Separator)

func removeSegment(p string) string {
	prefix := ""
	minParts := 2
	if strings.HasPrefix(p, "/") {
		prefix = "/"
		minParts = 1
	}

	parts := strings.FieldsFunc(p, func(r rune) bool {
		switch r {
		default:
			return false
		case '/', '\\':
			return true
		}
	})
	if len(parts) <= minParts {
		return ""
	}
	return prefix + strings.Join(parts[:len(parts)-1], filepathSeparator)
}
