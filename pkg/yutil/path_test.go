package yutil_test

import (
	"testing"

	"github.com/Mirantis/rekustomize/pkg/yutil"
)

func TestPathString(t *testing.T) {
	tests := map[string]struct {
		input yutil.NodePath
		want  string
	}{
		`nil`: {
			nil,
			"/",
		},
		`empty`: {
			yutil.NodePath{},
			"/",
		},
		`simple`: {
			yutil.NodePath{"a", "b", "c"},
			"/a/b/c",
		},
		`escaped "/"`: {
			yutil.NodePath{"a/b", "c"},
			"/a~1b/c",
		},
		`escaped "~"`: {
			yutil.NodePath{"a~b", "c"},
			"/a~0b/c",
		},
		`escaped "~" and "/"`: {
			yutil.NodePath{"a~b", "c/d", "e~/~f", "g/~/h"},
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
