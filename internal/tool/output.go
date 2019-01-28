package tool

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type HandleOutput func(ctx context.Context, filename string, content []byte) error

var outputRegistry = map[string]HandleOutput{}

func RegisterOutputHandler(root string, h HandleOutput) {
	_, ok := outputRegistry[root]
	if ok {
		panic(fmt.Errorf("output handler %q already exists", root))
	}
	outputRegistry[root] = h
}

func outputFile(ctx context.Context, to string, filename string, content []byte) error {
	if h, ok := outputRegistry[to]; ok {
		return h(ctx, filename, content)
	}
	_ = os.MkdirAll(to, 0700)
	p := filepath.Join(to, filename)
	return ioutil.WriteFile(p, content, 0600)
}
