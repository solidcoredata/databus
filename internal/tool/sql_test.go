package tool_test

import (
	"bytes"
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kardianos/task"
	"solidcoredata.org/src/databus/internal/tool"
)

func testdata() string {
	_, fullTestFile, _, _ := runtime.Caller(0)
	dir, _ := filepath.Split(fullTestFile)
	return filepath.Join(dir, "testdata")
}

func runcmd(t *testing.T, subdir []string, args []string) {
	cmd := tool.BusCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	st := &task.State{
		Env:    []string{},
		Dir:    filepath.Join(testdata(), filepath.Join(subdir...)),
		Stdout: stdout,
		Stderr: stderr,
		ErrorLogger: func(err error) {
			t.Fatal(err)
		},
		MsgLogger: func(msg string) {
			t.Log(msg)
		},
	}
	err := task.ScriptAdd(cmd.Exec(args)).Run(context.Background(), st, nil)
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
	runcmd(t, []string{"library"}, []string{"validate"})
}
