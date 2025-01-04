package types_test

import (
	"testing"

	"github.com/Mirantis/rekustomize/pkg/types"
)

func TestPathString(t *testing.T) {
	tests := map[string]struct {
		input types.NodePath
		want  string
	}{
		`nil`: {
			nil,
			"/",
		},
		`empty`: {
			types.NodePath{},
			"/",
		},
		`simple`: {
			types.NodePath{"a", "b", "c"},
			"/a/b/c",
		},
		`escaped "/"`: {
			types.NodePath{"a/b", "c"},
			"/a~1b/c",
		},
		`escaped "~"`: {
			types.NodePath{"a~b", "c"},
			"/a~0b/c",
		},
		`escaped "~" and "/"`: {
			types.NodePath{"a~b", "c/d", "e~/~f", "g/~/h"},
			"/a~0b/c~1d/e~0~1~0f/g~1~0~1h",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := test.input.String()
			if got != test.want {
				t.Errorf("mismatch for path %#v: got %#v, want %#v", test.input, got, test.want)
			}
		})
	}
}
