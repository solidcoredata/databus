package tool

import (
	"context"

	"solidcoredata.org/src/databus/bus"
)

var _ bus.RunStart = &SQLGenerate{}

type SQLGenerate struct{}

func (s *SQLGenerate) NodeTypes(ctx context.Context, header *bus.CallHeader, request *bus.CallNodeTypesRequest) (*bus.CallNodeTypesResponse, error) {
	return &bus.CallNodeTypesResponse{
		CallVersion: 1,
		NodeTypes: []string{
			"solidcoredata.org/type/sql/table",
		},
	}, nil
}

func (s *SQLGenerate) Run(ctx context.Context, header *bus.CallHeader, request *bus.CallRunRequest) (*bus.CallRunResponse, error) {
	c := request.Current
	_ = c
	// TODO(daniel.theophanes): produce an output, send to an http server (started on test system), that verifies it matches golden file.
	return &bus.CallRunResponse{
		CallVersion: 1,
	}, nil
}
