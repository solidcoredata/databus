package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"solidcoredata.org/src/databus/bus"
	"solidcoredata.org/src/databus/bus/sysfs"

	"github.com/kardianos/task"
)

func BusCommand() *task.Command {
	fProject := &task.Flag{Name: "project", Type: task.FlagString, Default: "", Usage: "Project directory, if empty, uses current working directory."}
	fSrc := &task.Flag{Name: "src", Type: task.FlagBool, Default: false, Usage: "True if the src should be used as the current version and the most recent checkin the previous version."}

	cmd := &task.Command{
		Flags: []*task.Flag{fProject},
		Usage: fmt.Sprintf(`Solid Core Data Bus

The root of the data bus project is defined by a %q file.
Tasks set to run are defined in this file as well.
The source for the current input bus should live under %q.
`, sysfs.ConfigFilename, filepath.Join(sysfs.InputDir, sysfs.InputFilename)),
		Commands: []*task.Command{
			{
				Name:  "validate",
				Usage: "Validate the data bus.",
				Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
					project := st.Default(fProject.Name, "").(string)
					return validate(ctx, st.Filepath(project))
				}),
			},
			{
				Name:  "diff",
				Usage: "Show the current diff between the current src data bus and current bus.",
				Flags: []*task.Flag{fSrc},
				Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
					project := st.Default(fProject.Name, "").(string)
					src := st.Default(fSrc.Name, false).(bool)
					return diff(ctx, st.Filepath(project), src, st.Stdout)
				}),
			},
			{
				Name:  "commit",
				Usage: "Commit the data bus as a new version.",
				Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
					project := st.Default(fProject.Name, "").(string)
					return commit(ctx, st.Filepath(project))
				}),
			},
			{
				Name:  "generate",
				Usage: "Generate the configured tasks on the data bus. Defaults to running on the last commited bus.",
				Flags: []*task.Flag{fSrc},
				Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
					project := st.Default(fProject.Name, "").(string)
					src := st.Default(fSrc.Name, false).(bool)
					return run(ctx, st.Filepath(project), src)
				}),
			},
			{
				Name:  "deploy",
				Usage: "Deploy the current configuration to a running system.",
				Flags: []*task.Flag{fSrc},
				Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
					return fmt.Errorf("deploy not implemented")
				}),
			},
			{
				Name:  "ui",
				Usage: "Show the development user interface.",
				Flags: []*task.Flag{fSrc},
				Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
					return fmt.Errorf("ui not implemented")
				}),
			},
		},
	}
	return cmd
}

// validate looks for the root definition, loads it,
// then validates it for basic correctness.
func validate(ctx context.Context, projectPath string) error {
	root, err := sysfs.RootFromWD(projectPath)
	if err != nil {
		return err
	}
	sys := sysfs.NewSystem(root)
	b, err := sys.ReadBus(ctx, bus.InputOptions{Src: true})
	if err != nil {
		return err
	}

	err = b.Init()
	if err != nil {
		return err
	}
	return nil
}

func run(ctx context.Context, projectPath string, src bool) error {
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

	err = bc.Init()
	if err != nil {
		return err
	}
	err = bp.Init()
	if err != nil {
		return err
	}

	delta, err := bus.NewDelta(bc, bp)
	if err != nil {
		return err
	}

	err = sys.Run(ctx, project, bc, bp, delta)
	if err != nil {
		return err
	}

	return nil
}

func diff(ctx context.Context, projectPath string, src bool, out io.Writer) error {
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

	err = bc.Init()
	if err != nil {
		return err
	}
	err = bp.Init()
	if err != nil {
		return err
	}

	delta, err := bus.NewDelta(bc, bp)
	if err != nil {
		return err
	}

	encode := json.NewEncoder(out)
	encode.SetEscapeHTML(false)
	encode.SetIndent("", "\t")
	return encode.Encode(delta)
}

func commit(ctx context.Context, projectPath string) error {
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

	err = bc.Init()
	if err != nil {
		return err
	}
	err = bp.Init()
	if err != nil {
		return err
	}

	delta, err := bus.NewDelta(bc, bp)
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
