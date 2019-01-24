package tool

import (
	"context"

	"solidcoredata.org/src/databus/bus"
	"solidcoredata.org/src/databus/internal/start"
)

var _ start.RunStart = &SQLGenerate{}

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
	return &bus.CallRunResponse{
		CallVersion: 1,
	}, nil
}
