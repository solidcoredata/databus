package main

import (
	"fmt"
	"testing"
)

func TestRemoveSegment(t *testing.T) {
	list := []struct {
		In  string
		Out string
		Sep string
	}{
		{"/abc/123/456", "/abc/123", "/"},
		{"/abc/123", "/abc", "/"},
		{`C:\project\src\foo`, `C:\project\src`, `\`},
		{`C:\project\src`, `C:\project`, `\`},
		{`C:\project`, ``, `\`},
	}
	for index, item := range list {
		t.Run(fmt.Sprintf("row-%d", index+1), func(t *testing.T) {
			filepathSeparator = item.Sep
			want := item.Out
			got := removeSegment(item.In)
			if got != want {
				t.Fatalf("got %q want %q", got, want)
			}
		})
	}
}
