package tool

import (
	"context"

	"solidcoredata.org/src/databus/bus"
	"solidcoredata.org/src/databus/internal/start"
)

var _ start.RunStart = &SQLGenerate{}

type SQLGenerate struct{}

func (s *SQLGenerate) NodeTypes(ctx context.Context, header *bus.CallHeader, request *bus.CallNodeTypesRequest) (*bus.CallNodeTypesResponse, error) {
	return &bus.CallNodeTypesResponse{}, nil
}

func (s *SQLGenerate) Run(ctx context.Context, header *bus.CallHeader, request *bus.CallRunRequest) (*bus.CallRunResponse, error) {
	return &bus.CallRunResponse{}, nil
}
