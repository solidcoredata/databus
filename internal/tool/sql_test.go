package tool_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kardianos/task"
	"solidcoredata.org/src/databus/bus/sysfs"
	"solidcoredata.org/src/databus/internal/tool"
)

func init() {
	sysfs.RegisterMemoryRunner("memory://run/tool/sql", &tool.SQLGenerate{})
	tool.RegisterOutputHandler("memory://verify/output", verifyOutput)
}

func testdata() string {
	_, fullTestFile, _, _ := runtime.Caller(0)
	dir, _ := filepath.Split(fullTestFile)
	return filepath.Join(dir, "testdata")
}

func verifyOutput(ctx context.Context, filename string, content []byte) error {
	p := filepath.Join(dirFromContext(ctx), "output", filename)
	// Read file at p.
	// Compare to content.
	golden, err := ioutil.ReadFile(p)
	if err != nil {
		return err
	}
	if !bytes.Equal(golden, content) {
		return fmt.Errorf("%s does not match:\n%s", filename, content)
	}
	return nil
}

type ctxKeyDir struct{}

func dirFromContext(ctx context.Context) string {
	return ctx.Value(ctxKeyDir{}).(string)
}
func dirWithContext(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, ctxKeyDir{}, dir)
}

func runcmd(t *testing.T, subdir []string, args []string) {
	ctx := context.Background()
	wd := filepath.Join(testdata(), filepath.Join(subdir...))
	cmd := tool.BusCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	ctx = dirWithContext(ctx, wd)

	st := &task.State{
		Env:    map[string]string{},
		Dir:    wd,
		Stdout: stdout,
		Stderr: stderr,
		ErrorLogger: func(err error) {
			t.Fatal(err)
		},
		MsgLogger: func(msg string) {
			t.Log(msg)
		},
	}
	err := task.ScriptAdd(cmd.Exec(args)).Run(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	if stderr.Len() > 0 {
		t.Fatal(stderr.String())
	}
	if stdout.Len() > 0 {
		t.Log(stdout.String())
	}
}

func TestSQL(t *testing.T) {
	runcmd(t, []string{"library"}, []string{"run", "-src"})
}
