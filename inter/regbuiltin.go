package inter

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

var _ ExtensionRegister = &Builtin{}

func NewBuiltinExtentionRegister() *Builtin {
	return &Builtin{
		exts: make(map[string]Extension),
	}
}

type Builtin struct {
	mu   sync.RWMutex
	exts map[string]Extension
}

func (b *Builtin) Add(ctx context.Context, exts ...Extension) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	add := make(map[string]Extension, len(exts))
	for _, e := range exts {
		info := e.AboutSelf()

		_, ok := add[info.Name]
		if ok {
			return fmt.Errorf("cannot add extension type %q twice", info.Name)
		}
		_, ok = b.exts[info.Name]
		if ok {
			return fmt.Errorf("extension %q already registered", info.Name)
		}
		add[info.Name] = e
	}
	for name, e := range add {
		b.exts[name] = e
	}
	return nil
}

func (b *Builtin) List(ctx context.Context) ([]string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	list := make([]string, 0, len(b.exts))
	for name := range b.exts {
		list = append(list, name)
	}
	sort.Strings(list)

	return list, nil
}
func (b *Builtin) Get(ctx context.Context, name string) (Extension, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	e, ok := b.exts[name]
	if !ok {
		return nil, fmt.Errorf("extension %q not found", name)
	}

	return e, nil
}
