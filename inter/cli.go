package inter

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kardianos/task"
	"solidcoredata.org/src/databus/bus/sysfs"
)

func projectRoot(explicitRoot string) (string, error) {
	root := explicitRoot
	lookForRoot := false
	if len(root) == 0 {
		lookForRoot = true
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if lookForRoot {
		// TODO(danielt.theophanes): look for the root project file.
	}

	return root, nil
}

func RunCLI() error {
	fProject := &task.Flag{Name: "project", Type: task.FlagString, Default: "", Usage: "Project directory, if empty, uses current working directory."}
	fSrc := &task.Flag{Name: "src", Type: task.FlagBool, Default: false, Usage: "True if the src should be used as the current version and the most recent checkin the previous version."}

	return task.Start(context.Background(), time.Second*2, func(ctx context.Context) error {
		extReg := NewBuiltinExtentionRegister()
		extCRDB := NewCRDB()
		err := extReg.Add(ctx, extCRDB)
		if err != nil {
			return err
		}

		setupSystem := func(project string) (*SimpleCaller, error) {
			root, err := projectRoot(project)
			if err != nil {
				return nil, err
			}
			fb, err := NewFileBus(root)
			if err != nil {
				return nil, err
			}
			rwext, err := NewFileExtRW(root)
			if err != nil {
				return nil, err
			}
			return NewCaller(CallerSetup{
				Bus:       fb,
				Versioner: fb,
				ExtRW:     rwext,
				ExtReg:    extReg,
			})
		}

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
						caller, err := setupSystem(project)
						if err != nil {
							return err
						}
						return caller.Validate(ctx)
					}),
				},
				{
					Name:  "diff",
					Usage: "Show the current diff between the current src data bus and current bus.",
					Flags: []*task.Flag{fSrc},
					Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
						project := st.Default(fProject.Name, "").(string)
						src := st.Default(fSrc.Name, false).(bool)
						caller, err := setupSystem(project)
						if err != nil {
							return err
						}
						diff, err := caller.Diff(ctx, src)
						if err != nil {
							return err
						}
						st.Log(diff.String())
						return nil
					}),
				},
				{
					Name:  "commit",
					Usage: "Commit the data bus as a new version.",
					Flags: []*task.Flag{
						{Name: "amend", Type: task.FlagBool, Default: false, Usage: "Revise the most recent commit."},
					},
					Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
						project := st.Default(fProject.Name, "").(string)
						amend := st.Default("amend", false).(bool)
						caller, err := setupSystem(project)
						if err != nil {
							return err
						}
						ver, err := caller.Commit(ctx, amend)
						if err != nil {
							return err
						}
						st.Logf("Version: %d-%s", ver.Sequence, hex.EncodeToString(ver.Identifier[:]))
						return nil
					}),
				},
				{
					Name:  "generate",
					Usage: "Generate the configured tasks on the data bus. Defaults to running on the last commited bus.",
					Flags: []*task.Flag{fSrc},
					Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
						project := st.Default(fProject.Name, "").(string)
						src := st.Default(fSrc.Name, false).(bool)
						caller, err := setupSystem(project)
						if err != nil {
							return err
						}
						return caller.Generate(ctx, src)
					}),
				},
				{
					Name:  "deploy",
					Usage: "Deploy the current configuration to a running system.",
					Flags: []*task.Flag{
						fSrc,
						{Name: "create", Type: task.FlagBool, Default: false, Usage: ""},
						{Name: "delete", Type: task.FlagBool, Default: false, Usage: ""},
					},
					Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
						project := st.Default(fProject.Name, "").(string)
						src := st.Default(fSrc.Name, false).(bool)
						caller, err := setupSystem(project)
						if err != nil {
							return err
						}
						tasks := st.Get("args").([]string)
						opts := &DeployOptions{
							CreateEnvironment: st.Default("create", false).(bool),
							RunTasks:          tasks,
							DeleteEnvironment: st.Default("delete", false).(bool),
						}
						return caller.Deploy(ctx, src, opts)
					}),
				},
				{
					Name:  "ui",
					Usage: "Show the development user interface.",
					Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
						project := st.Default(fProject.Name, "").(string)
						caller, err := setupSystem(project)
						if err != nil {
							return err
						}
						return caller.UI(ctx)
					}),
				},
			},
		}

		st := task.DefaultState()
		return cmd.Exec(os.Args[1:]).Run(ctx, st, nil)
	})
}
