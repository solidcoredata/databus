package start

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"solidcoredata.org/src/databus/bus"
)

type RunStart interface {
	NodeTypes(ctx context.Context, header *bus.CallHeader, request *bus.CallNodeTypesRequest) (*bus.CallNodeTypesResponse, error)
	Run(ctx context.Context, header *bus.CallHeader, request *bus.CallRunRequest) (*bus.CallRunResponse, error)
}

func Run(ctx context.Context, rs RunStart) error {
	return Start(ctx, time.Second*1, func(ctx context.Context) error {
		decode := json.NewDecoder(os.Stdin)
		decode.DisallowUnknownFields()
		decode.UseNumber()

		encode := json.NewEncoder(os.Stdout)
		encode.SetEscapeHTML(false)
		encode.SetIndent("", "\t")

		header := &bus.CallHeader{}
		err := decode.Decode(header)
		if err != nil {
			return err
		}
		switch header.Type {
		default:
			return fmt.Errorf("unknown type %q", header.Type)
		case "NodeTypes":
			req := &bus.CallNodeTypesRequest{}
			err = decode.Decode(req)
			if err != nil {
				return err
			}
			resp, err := rs.NodeTypes(ctx, header, req)
			if err != nil {
				return err
			}
			return encode.Encode(resp)

		case "Run":
			req := &bus.CallRunRequest{}
			err = decode.Decode(req)
			if err != nil {
				return err
			}
			resp, err := rs.Run(ctx, header, req)
			if err != nil {
				return err
			}
			return encode.Encode(resp)

		}
	})
}
