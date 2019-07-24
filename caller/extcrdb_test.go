package caller

import (
	"bytes"
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"

	"solidcoredata.org/src/databus/bus"
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

func TestSQL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	extcrdb := NewCRDB()
	about := extcrdb.AboutSelf()

	root := filepath.Join(testdata(), "library")
	loader, err := NewFileBus(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := loader.GetBus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	delta, err := bus.NewDelta(b.Filter(about.HandleTypes), nil)
	if err != nil {
		t.Fatal(err)
	}

	err = extcrdb.Generate(ctx, delta, func(ctx context.Context, filename string, content []byte) error {
		t.Run("run SQL", func(t *testing.T) {
			// If crdb is available, try to create the database script.
			crdb, err := exec.LookPath("cockroach")
			if err != nil {
				t.Skip("cockroach executable not found")
			}
			ctx, stop := context.WithCancel(ctx)
			defer stop()

			w := &waitForMsg{Msg: []byte("nodeID:")}
			cmd := exec.CommandContext(ctx, crdb, strings.Split("start --insecure --listen-addr=localhost:9999 --store=type=mem,size=1GiB --logtostderr=NONE", " ")...)
			cmd.Stdout = w
			err = cmd.Start()
			if err != nil {
				t.Fatalf("unable to run cockroach: %v", err)
			}
			w.Wait()
			connector, err := pq.NewConnector("postgres://root@localhost:9999?sslmode=disable")
			if err != nil {
				t.Fatal(err)
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
				t.Fatalf("failed to ping database: %v", err)
			}

			pool.ExecContext(ctx, "drop database library;")
			_, err = pool.ExecContext(ctx, string(content))
			if err != nil {
				switch err := err.(type) {
				case *pq.Error:
					t.Fatalf("%#v\n\n%s", err, content)
				default:
					t.Fatalf("%v\n\n%s", err, content)
				}
			}

			cmd.Process.Kill()
		})
		t.Run("compare SQL", func(t *testing.T) {
			// Now compare the generated SQL to the golden sql.``
			p := filepath.Join(root, "output", filename)
			pCheck := filepath.Join(root, "output", "ck_"+filename)
			// Read file at p.
			// Compare to content.
			golden, err := ioutil.ReadFile(p)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			if !bytes.Equal(golden, content) {
				ioutil.WriteFile(pCheck, content, 0600)
				t.Fatalf("%s does not match:\n%s", filename, content)
			}
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
