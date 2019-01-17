package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"

	"solidcoredata.org/src/databus/bus"
	"solidcoredata.org/src/databus/bus/analysis"
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
	p := &program{}

	fProject := &task.Flag{Name: "project", Type: task.FlagString, Default: "", Usage: "Project directory, if empty, uses current working directory."}
	fSrc := &task.Flag{Name: "src", Type: task.FlagBool, Default: false, Usage: "True if the src should be used as the current version and the most recent checkin the previous version."}

	cmd := &task.Command{
		Flags: []*task.Flag{fProject},
		Usage: `Solid Core Data Bus

The root of the data bus project is defined by a "` + sysfs.ConfigFilename + `" file.
Tasks set to run are defined in this file as well.
The source for the current input bus should live under "` + filepath.Join(sysfs.InputDir, sysfs.InputFilename) + `".
`,
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
				Flags: []*task.Flag{fSrc},
				Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
					project := st.Default(fProject.Name, "").(string)
					src := st.Default(fSrc.Name, false).(bool)
					return p.diff(ctx, st.Filepath(project), src, st.Stdout)
				}),
			},
			{
				Name:  "commit",
				Usage: "Commit the data bus as a new version.",
				Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
					project := st.Default(fProject.Name, "").(string)
					return p.commit(ctx, st.Filepath(project))
				}),
			},
			{
				Name:  "run",
				Usage: "Run the configured tasks on the data bus. Defaults to running on the last commited bus.",
				Flags: []*task.Flag{fSrc},
				Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
					project := st.Default(fProject.Name, "").(string)
					src := st.Default(fSrc.Name, false).(bool)
					return p.run(ctx, st.Filepath(project), src)
				}),
			},
		},
	}

	st := task.DefaultState()
	err := cmd.Exec(os.Args[1:]).Run(context.Background(), st, nil)
	if err != nil {
		log.Fatal(err)
	}
}

type program struct{}

// validate looks for the root definition, loads it,
// then validates it for basic correctness.
func (p *program) validate(ctx context.Context, projectPath string) error {
	root, err := sysfs.RootFromWD(projectPath)
	if err != nil {
		return err
	}
	sys := sysfs.NewSystem(root)
	b, err := sys.ReadBus(ctx, bus.InputOptions{Src: true})
	if err != nil {
		return err
	}

	_, err = analysis.New(ctx, b)
	if err != nil {
		return err
	}
	return nil
}

func (p *program) run(ctx context.Context, projectPath string, src bool) error {
	root, err := sysfs.RootFromWD(projectPath)
	if err != nil {
		return err
	}
	sys := sysfs.NewSystem(root)

	project, err := sys.ReadProject(ctx)
	if err != nil {
		return err
	}

	ioc := bus.InputOptions{}
	iop := bus.InputOptions{}
	if src {
		ioc.Src = true
		iop.Version = 0
	} else {
		ioc.Version = 0
		iop.Version = -1
	}
	bc, err := sys.ReadBus(ctx, ioc)
	if err != nil {
		return err
	}
	bp, err := sys.ReadBus(ctx, iop)
	if err != nil {
		bp = &bus.Bus{}
	}

	ac, err := analysis.New(ctx, bc)
	if err != nil {
		return err
	}
	ap, err := analysis.New(ctx, bp)
	if err != nil {
		return err
	}

	delta, err := analysis.NewDelta(ac, ap)
	if err != nil {
		return err
	}

	err = sys.Run(ctx, project, bc, bp, delta)
	if err != nil {
		return err
	}

	return nil
}

func (p *program) diff(ctx context.Context, projectPath string, src bool, out io.Writer) error {
	root, err := sysfs.RootFromWD(projectPath)
	if err != nil {
		return err
	}
	sys := sysfs.NewSystem(root)

	ioc := bus.InputOptions{}
	iop := bus.InputOptions{}
	if src {
		ioc.Src = true
		iop.Version = 0
	} else {
		ioc.Version = 0
		iop.Version = -1
	}
	bc, err := sys.ReadBus(ctx, ioc)
	if err != nil {
		return err
	}
	bp, err := sys.ReadBus(ctx, iop)
	if err != nil {
		bp = &bus.Bus{}
	}

	ac, err := analysis.New(ctx, bc)
	if err != nil {
		return err
	}
	ap, err := analysis.New(ctx, bp)
	if err != nil {
		return err
	}

	delta, err := analysis.NewDelta(ac, ap)
	if err != nil {
		return err
	}

	encode := json.NewEncoder(out)
	encode.SetEscapeHTML(false)
	encode.SetIndent("", "\t")
	return encode.Encode(delta)
}

func (p *program) commit(ctx context.Context, projectPath string) error {
	root, err := sysfs.RootFromWD(projectPath)
	if err != nil {
		return err
	}
	sys := sysfs.NewSystem(root)

	ioc := bus.InputOptions{Src: true}
	iop := bus.InputOptions{Version: -1}

	bc, err := sys.ReadBus(ctx, ioc)
	if err != nil {
		return err
	}
	bp, err := sys.ReadBus(ctx, iop)
	if err != nil {
		bp = &bus.Bus{}
	}
	bc.Version.Version = bp.Version.Version + 1

	ac, err := analysis.New(ctx, bc)
	if err != nil {
		return err
	}
	ap, err := analysis.New(ctx, bp)
	if err != nil {
		return err
	}

	delta, err := analysis.NewDelta(ac, ap)
	if err != nil {
		return err
	}
	err = sys.WriteBus(ctx, bc)
	if err != nil {
		return err
	}
	err = sys.WriteDelta(ctx, delta)
	if err != nil {
		return err
	}
	return nil
}
