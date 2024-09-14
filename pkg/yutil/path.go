package yutil

import "strings"

var rfc6901replacer = strings.NewReplacer("~", "~0", "/", "~1")

// Path represents YAML node path
type Path []string

// String returns a RFC6901-escaped representation of the path
func (p Path) String() string {
	escaped := make([]string, 0, len(p))
	for _, s := range p {
		escaped = append(escaped, rfc6901replacer.Replace(s))
	}
	return "/" + strings.Join(escaped, "/")
}
