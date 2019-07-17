package inter

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

	"github.com/lib/pq"
)

// Start a server listening for SQL connections on localhost:9999.
// cockroach start --insecure --listen-addr=localhost:9999 --store=type=mem,size=1GiB --logtostderr=NONE

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

type testroot string

func (tr testroot) verifyOutput(ctx context.Context, filename string, content []byte) error {
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

		for i := 0; i < 30; i++ {
			err = pool.PingContext(ctx)
			if err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if err != nil {
			return fmt.Errorf("failed to ping database: %v", err)
		}

		_, err = pool.ExecContext(ctx, string(content))
		if err != nil {
			return err
		}
		cmd.Process.Kill()
	}

	// Now compare the generated SQL to the golden sql.``
	p := filepath.Join(string(tr), "output", filename)
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

func TestSQL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	root := filepath.Join(testdata(), "library")
	loader, err := NewFileBus(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := loader.GetBus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = b.Init()
	if err != nil {
		t.Fatal(err)
	}

	extcrdb := NewCRDB()
	about, err := extcrdb.AboutSelf(ctx)
	if err != nil {
		t.Fatal(err)
	}
	bfilter := b.Filter(about.HandleTypes)
	err = extcrdb.Generate(ctx, bfilter, testroot(root).verifyOutput)
	if err != nil {
		t.Fatal(err)
	}
}
