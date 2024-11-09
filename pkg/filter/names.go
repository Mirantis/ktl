package filter

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

func parsePatterns(patterns []string) (include, exclude []string, err error) {
	errs := []error{}

	for _, rawPattern := range patterns {
		pattern, isExclude := strings.CutPrefix(rawPattern, "!")
		if _, err := path.Match(pattern, ""); err != nil {
			errs = append(errs, fmt.Errorf("%v: %q", err, rawPattern))
			continue
		}
		if isExclude {
			exclude = append(exclude, pattern)
		} else {
			include = append(include, pattern)
		}
	}

	if err := errors.Join(errs...); err != nil {
		return nil, nil, err
	}

	return include, exclude, nil
}

func matchPatterns(name string, patterns []string) bool {
	for _, pattern := range patterns {
		if ok, _ := path.Match(pattern, name); ok {
			return true
		}
	}
	return false
}

func SelectNames(from, patterns []string) ([]string, error) {
	result := make([]string, 0, len(from))
	include, exclude, err := parsePatterns(patterns)
	if err != nil {
		return nil, err
	}

	for _, name := range from {
		if matchPatterns(name, exclude) {
			continue
		}
		if len(include) > 0 && !matchPatterns(name, include) {
			continue
		}
		result = append(result, name)
	}

	return result, nil
}
