package sysfs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"solidcoredata.org/src/databus/bus"
)

var memoryReg = map[string]bus.RunStart{}

func RegisterMemoryRunner(name string, r bus.RunStart) {
	_, exist := memoryReg[name]
	if exist {
		panic(fmt.Errorf("run start %q already exists in registry", name))
	}
	memoryReg[name] = r
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
		wd = filepath.Join(r.root, "run")
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
		// TODO(daniel.theophanes): Filter bus nodes in bus and delta to only include requested node types.
		runReq := &bus.CallRunRequest{
			CallVersion: ntResp.CallVersion,
			Root:        wd,

			Current:  currentBus,
			Previous: previousBus,
			DeltaBus: deltaBus,
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
	}
	if errs != nil {
		return errs
	}
	return nil
}

func (r *runner) runExec(ctx context.Context, s stdbuf, call string, inheader *bus.CallHeader, inObj, outObj interface{}) error {
	if strings.HasPrefix(call, "memory://") {
		rs, ok := memoryReg[call]
		if !ok {
			return fmt.Errorf("call %q not found in registry", call)
		}
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
