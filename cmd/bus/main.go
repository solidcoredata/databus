package main

import (
	"context"
	"log"
	"os"

	"solidcoredata.org/src/databus/bus"

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

   /bus/input
   /bus/output
   /run
       /runner-name.com/sql
       /runner-name.com/ui
*/
func main() {
	p := &program{
		analysis: &bus.Analysis{},
	}

	fBus := &task.Flag{Name: "bus", Type: task.FlagString, Default: "bus.jsonnet", Usage: "File name of the bus definition, may be json or jsonnet."}

	cmd := &task.Command{
		Usage: `Solid Core Data Bus

The root of the data bus project is defined by a "X" file.
Tasks are run defined in "Y" file.`,
		Commands: []*task.Command{
			{
				Name:  "validate",
				Usage: "Validate the data bus.",
				Flags: []*task.Flag{fBus},
				Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
					busName := st.Default(fBus.Name, "").(string)
					return p.validate(ctx, st.Filepath(busName))
				}),
			},
			{
				Name:  "checkpoint",
				Usage: "Checkpoint the data bus as a new version.",
			},
			{
				Name:  "run",
				Usage: "Run the configured tasks on the data bus.",
			},
		},
	}

	st := task.DefaultState()
	err := cmd.Exec(os.Args[1:]).Run(context.Background(), st, nil)
	if err != nil {
		log.Fatal(err)
	}
}

type program struct {
	analysis *bus.Analysis
}

// validate looks for the root definition, loads it,
// then validates it for basic correctness.
func (p *program) validate(ctx context.Context, busPath string) error {
	b, err := bus.LoadBus(ctx, busPath)
	if err != nil {
		return err
	}
	return p.analysis.Validate(ctx, b)
}
