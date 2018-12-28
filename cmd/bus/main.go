package main

import (
	"context"
	"log"
	"os"

	"solidcoredata.org/src/databus/bus"

	"github.com/kardianos/task"
)

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
	bus, err := p.analysis.LoadBus(ctx, busPath)
	if err != nil {
		return err
	}
	return p.analysis.Validate(ctx, bus)
}
