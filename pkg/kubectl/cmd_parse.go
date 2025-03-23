package kubectl

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type parserFn[T any] func([]byte) (T, error)

func parseNoop(_ []byte) (bool, error) {
	return true, nil
}

func parseRNodes(data []byte) ([]*yaml.RNode, error) {
	kreader := &kio.ByteReader{Reader: bytes.NewBuffer(data)}

	return kreader.Read() //nolint:wrapcheck
}

func parseLines(data []byte) ([]string, error) {
	lines := []string{}

	s := bufio.NewScanner(bytes.NewBuffer(data))
	for s.Scan() {
		lines = append(lines, s.Text())
	}

	return lines, nil
}

func parseResNames(data []byte) ([]string, error) {
	names, err := parseLines(data)
	if err != nil {
		return nil, err
	}

	for idx := range names {
		sepIdx := strings.LastIndex(names[idx], "/")
		if sepIdx < 0 {
			continue
		}

		names[idx] = names[idx][sepIdx+1:]
	}

	return names, nil
}

func jsonParser[T any](dst T, def T) parserFn[T] {
	return func(data []byte) (T, error) {
		if err := json.Unmarshal(data, dst); err != nil {
			return def, err //nolint:wrapcheck
		}

		return dst, nil
	}
}
