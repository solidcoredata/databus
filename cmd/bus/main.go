package main

import (
	"context"
	"log"
	"os"
	"time"

	"solidcoredata.org/src/databus/internal/tool"

	"github.com/kardianos/task"
)

/*
   There are three types of folder areas:
    * Input to create a Bus.
    * Bus output / current, versioned, and diff.
    * Run output / current, alter.

   Start with defining these folders.
   Then only work on the "current" version (no alters, no versions).

   For now, let's make the project structure ridgid, and
   have defined areas for each input and output.
   For testing and future options, make all output relative to a virtual
   root in a virtual filesystem.
   Each sub-system will write to a sub-vfs into a virual root folder.
*/

func main() {
	cmd := tool.BusCommand()
	err := task.Start(context.Background(), time.Second*2, func(ctx context.Context) error {
		st := task.DefaultState()
		return cmd.Exec(os.Args[1:]).Run(context.Background(), st, nil)
	})
	if err != nil {
		log.Fatal(err)
	}
}
