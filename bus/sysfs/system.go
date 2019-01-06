package sysfs

import (
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

// RootFromWD starts in the current working directory and looks for the root
// of the bus directory. If not found it returns an error.
func RootFromWD() (string, error) {
	panic("todo")
}
