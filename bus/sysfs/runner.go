package sysfs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"solidcoredata.org/src/databus/bus"
)

type memoryType struct {
	mu     sync.Mutex
	files  map[string][]byte
	rs     bus.RunStart
	verify verifyFunc
}
type verifyFunc func(ctx context.Context, filename string, content []byte) error

var memoryReg = map[string]*memoryType{}

func RegisterMemoryRunner(name string, r bus.RunStart, verify verifyFunc) {
	_, exist := memoryReg[name]
	if exist {
		panic(fmt.Errorf("run start %q already exists in registry", name))
	}
	memoryReg[name] = &memoryType{
		files:  make(map[string][]byte),
		rs:     r,
		verify: verify,
	}
}

var _ bus.Runner = &runner{}

func NewRunner(root string) bus.Runner {
	return &runner{
		root: root,
	}
}

type runner struct {
	root string
}

type stdbuf struct {
	wd   string
	sin  *bytes.Buffer
	sout *bytes.Buffer
	serr *bytes.Buffer
}

func (s stdbuf) Reset() {
	s.sin.Reset()
	s.sout.Reset()
	s.serr.Reset()
}

const (
	headerNodeTypes = "NodeTypes"
	headerRun       = "Run"
)

func (r *runner) Run(ctx context.Context, setup bus.Project, currentBus *bus.Bus, previousBus *bus.Bus, deltaBus *bus.DeltaBus) error {
	wd := setup.Root
	var errs *bus.Errors
	if len(wd) == 0 {
		wd = filepath.Join(r.root, "generate")
	}
	s := stdbuf{
		wd:   wd,
		sin:  &bytes.Buffer{},
		sout: &bytes.Buffer{},
		serr: &bytes.Buffer{},
	}

	for _, e := range setup.Enteries {
		// Assume Call path is an exec, not an http call. Call exec.
		header := &bus.CallHeader{
			Type:    headerNodeTypes,
			Options: e.Options,
		}
		ntReq := &bus.CallNodeTypesRequest{}
		ntResp := &bus.CallNodeTypesResponse{}
		err := r.runExec(ctx, s, e.Call, header, ntReq, ntResp)
		if err != nil {
			errs = errs.Append(err)
			continue
		}
		if len(ntResp.Errors) > 0 {
			for _, msg := range ntResp.Errors {
				errs = errs.AppendMsg("%s: %s", e.Name, msg)
			}
			continue
		}
		// Check the CallVersion, ensure that it is being populated.
		if ntResp.CallVersion <= 0 {
			errs = errs.AppendMsg("%s: invalid call version %d, must be greater then zero", e.Name, ntResp.CallVersion)
			continue
		}
		if len(ntResp.NodeTypes) == 0 {
			errs = errs.AppendMsg("%s: invalid node types response, must require at least one node type", e.Name)
			continue
		}
		header.Type = headerRun
		runReq := &bus.CallRunRequest{
			CallVersion: ntResp.CallVersion,

			Current:  currentBus.Filter(ntResp.NodeTypes),
			Previous: previousBus.Filter(ntResp.NodeTypes),
			DeltaBus: deltaBus.Filter(ntResp.NodeTypes),
		}
		runResp := &bus.CallRunResponse{}
		err = r.runExec(ctx, s, e.Call, header, runReq, runResp)
		if err != nil {
			errs = errs.Append(err)
			continue
		}
		if len(runResp.Errors) > 0 {
			for _, msg := range runResp.Errors {
				errs = errs.AppendMsg("%s: %s", e.Name, msg)
			}
			continue
		}
		// Check the CallVersion, ensure that it is being populated.
		if ntResp.CallVersion != runResp.CallVersion {
			errs = errs.AppendMsg("%s: invalid call version %d, %d", e.Name, runResp.CallVersion, ntResp.CallVersion)
			continue
		}
		for _, file := range runResp.Files {
			if strings.HasPrefix(wd, "memory://") {
				mt, ok := memoryReg[e.Call]
				if !ok {
					return fmt.Errorf("call %q not found in registry", e.Call)
				}
				mt.mu.Lock()
				mt.files[file.Path] = file.Content
				mt.mu.Unlock()
				if mt.verify != nil {
					err = mt.verify(ctx, file.Path, file.Content)
					if err != nil {
						errs = errs.Append(err)
					}
				}
				continue
			}
			p := filepath.Join(wd, "output", e.Name, filepath.Clean(file.Path))
			pdir, _ := filepath.Split(p)
			err = os.MkdirAll(pdir, 0700)
			if err != nil {
				errs = errs.Append(err)
				break
			}
			err = ioutil.WriteFile(p, file.Content, 0600)
			if err != nil {
				errs = errs.Append(err)
			}
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

func (r *runner) runExec(ctx context.Context, s stdbuf, call string, inheader *bus.CallHeader, inObj, outObj interface{}) error {
	if strings.HasPrefix(call, "memory://") {
		mt, ok := memoryReg[call]
		if !ok {
			return fmt.Errorf("call %q not found in registry", call)
		}
		rs := mt.rs
		switch inheader.Type {
		default:
			return fmt.Errorf("unknown header type: %s", inheader.Type)
		case headerNodeTypes:
			in := inObj.(*bus.CallNodeTypesRequest)
			out := outObj.(*bus.CallNodeTypesResponse)
			resp, err := rs.NodeTypes(ctx, inheader, in)
			if err != nil {
				return err
			}
			*out = *resp
		case headerRun:
			in := inObj.(*bus.CallRunRequest)
			out := outObj.(*bus.CallRunResponse)
			resp, err := rs.Run(ctx, inheader, in)
			if err != nil {
				return err
			}
			*out = *resp
		}
		return nil
	}
	s.Reset()
	encode := json.NewEncoder(s.sin)
	encode.SetEscapeHTML(false)
	err := encode.Encode(inheader)
	if err != nil {
		return err
	}
	err = encode.Encode(inObj)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, call)
	cmd.Stdin = s.sin
	cmd.Stdout = s.sout
	cmd.Stderr = s.serr
	cmd.Dir = s.wd
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("%v, %s", err, s.serr.Bytes())
	}
	decode := json.NewDecoder(s.sout)
	decode.DisallowUnknownFields()
	err = decode.Decode(outObj)
	if err != nil {
		return err
	}
	return nil
}
