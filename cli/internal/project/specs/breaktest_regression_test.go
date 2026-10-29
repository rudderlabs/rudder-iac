package specs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRejectsAdditionalYAMLDocuments(t *testing.T) {
	data := []byte(`version: rudder/v1
kind: destination
metadata:
  name: first
spec:
  id: first
---
version: rudder/v1
kind: destination
metadata:
  name: second
spec:
  id: second
`)
	spec, err := New(data)
	require.Error(t, err)
	assert.ErrorContains(t, err, "multiple YAML documents")
	assert.Nil(t, spec)
}
