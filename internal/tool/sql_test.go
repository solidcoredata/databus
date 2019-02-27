package tool_test

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kardianos/task"
	"github.com/lib/pq"
	"solidcoredata.org/src/databus/bus/sysfs"
	"solidcoredata.org/src/databus/internal/tool"
)

// Start a server listening for SQL connections on localhost:9999.
// cockroach start --insecure --listen-addr=localhost:9999 --store=type=mem,size=1GiB --logtostderr=NONE

func init() {
	sysfs.RegisterMemoryRunner("memory://run/tool/sql", &tool.SQLGenerate{})
	tool.RegisterOutputHandler("memory://verify/output", verifyOutput)
}

func testdata() string {
	_, fullTestFile, _, _ := runtime.Caller(0)
	dir, _ := filepath.Split(fullTestFile)
	return filepath.Join(dir, "testdata")
}

type waitForMsg struct {
	lk  sync.Mutex
	Msg []byte

	done bool
}

func (w *waitForMsg) Wait() {
	w.lk.Lock()
}

func (w *waitForMsg) Write(b []byte) (int, error) {
	if w.done {
		return len(b), nil
	}
	// This is not entirely correct, but will likely work for this use case.
	if bytes.Contains(b, w.Msg) {
		w.done = true
		w.lk.Unlock()
	}
	return len(b), nil
}

func verifyOutput(ctx context.Context, filename string, content []byte) error {
	// If crdb is available, try to create the database script.
	crdb, err := exec.LookPath("cockroach")
	if err == nil {
		ctx, stop := context.WithCancel(ctx)
		defer stop()

		w := &waitForMsg{Msg: []byte("nodeID:")}
		cmd := exec.CommandContext(ctx, crdb, strings.Split("start --insecure --listen-addr=localhost:9999 --store=type=mem,size=1GiB --logtostderr=NONE", " ")...)
		cmd.Stdout = w
		err := cmd.Start()
		if err != nil {
			return fmt.Errorf("unable to run cockroach: %v", err)
		}
		w.Wait()
		connector, err := pq.NewConnector("postgres://root@localhost:9999?sslmode=disable")
		if err != nil {
			return err
		}
		pool := sql.OpenDB(connector)
		defer pool.Close()

		for i := 0; i < 10; i++ {
			err := pool.PingContext(ctx)
			if err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		_, err = pool.ExecContext(ctx, string(content))
		if err != nil {
			return err
		}
		cmd.Process.Kill()
	}

	// Now compare the generated SQL to the golden sql.``
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
