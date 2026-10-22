package v3

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type openAPIOperation struct {
	Security  []map[string][]string
	Responses map[string]struct {
		Content map[string]any
	}
}

func loadOpenAPIOperation(t *testing.T, path, method string) openAPIOperation {
	t.Helper()
	data, err := os.ReadFile("../../docs/v3-api.yaml")
	require.NoError(t, err)
	var spec struct {
		Paths map[string]map[string]yaml.Node
	}
	require.NoError(t, yaml.Unmarshal(data, &spec))
	require.Contains(t, spec.Paths, path)
	require.Contains(t, spec.Paths[path], method)
	var operation openAPIOperation
	node := spec.Paths[path][method]
	require.NoError(t, node.Decode(&operation))
	return operation
}
