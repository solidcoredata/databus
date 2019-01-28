package tool

import (
	"bytes"
	"context"
	"fmt"

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
	buf := &bytes.Buffer{}
	switch variant := header.Options["variant"]; variant {
	default:
		return nil, fmt.Errorf("unknown SQL variant type %q", variant)
	case "postgres":
		s.postgres(c, buf)
	}
	err := outputFile(ctx, request.Root, "schema.sql", buf.Bytes())
	if err != nil {
		return nil, err
	}
	return &bus.CallRunResponse{
		CallVersion: 1,
	}, nil
}

func (s *SQLGenerate) postgres(b *bus.Bus, buf *bytes.Buffer) {
	// TODO(daniel.theophanes): read properties and create schema, setup database, create tables, sort output by dependency order.
	buf.Write([]byte("-- This is pretend content."))
}
