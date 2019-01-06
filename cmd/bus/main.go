package main

import (
	"context"
	"log"
	"os"

	"solidcoredata.org/src/databus/bus"
	"solidcoredata.org/src/databus/bus/sysfs"

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
	p := &program{
		analysis: &bus.Analysis{},
	}

	fProject := &task.Flag{Name: "project", Type: task.FlagString, Default: "", Usage: "Project directory, if empty, uses current working directory."}

	cmd := &task.Command{
		Flags: []*task.Flag{fProject},
		Usage: `Solid Core Data Bus

The root of the data bus project is defined by a "X" file.
Tasks are run defined in "Y" file.`,
		Commands: []*task.Command{
			{
				Name:  "validate",
				Usage: "Validate the data bus.",
				Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
					project := st.Default(fProject.Name, "").(string)
					return p.validate(ctx, st.Filepath(project))
				}),
			},
			{
				Name:  "diff",
				Usage: "Show the current diff between the current src data bus and current bus.",
			},
			{
				Name:  "commit",
				Usage: "Commit the data bus as a new version.",
			},
			{
				Name:  "run",
				Usage: "Run the configured tasks on the data bus. Defaults to running on the last commited bus.",
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
func (p *program) validate(ctx context.Context, projectPath string) error {
	root, err := sysfs.RootFromWD(projectPath)
	if err != nil {
		return err
	}
	sys := sysfs.NewSystem(root)
	b, err := sys.ReadBus(ctx, bus.InputOptions{})
	if err != nil {
		return err
	}
	return p.analysis.Validate(ctx, b)
}
