package provider_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExamplesSync(t *testing.T) {
	// Initialize an empty backend
	b := backend.NewBackend()

	// Create a mock workspace
	workspace := &client.Workspace{
		ID:          "test-workspace-id",
		Name:        "Test Workspace",
		Environment: "DEVELOPMENT",
		Status:      "ACTIVE",
		Region:      "US",
	}

	var runSync = func(t *testing.T, specs map[string]string) error {
		// Create an example provider with that backend
		provider := example.NewProvider(b)

		// Load specs from testdata directory
		proj := project.New("dummy/path", provider, project.WithLoader(&mockLoader{specs: specs}))
		err := proj.Load()
		require.NoError(t, err, "Failed to load project specs")
		// Create syncer and sync the project resource graph to the backend
		targetGraph, err := proj.ResourceGraph()
		require.NoError(t, err, "Failed to get resource graph")
		s, err := syncer.New(provider, workspace)
		require.NoError(t, err, "Failed to create syncer")

		err = s.Sync(context.Background(), targetGraph)
		return err
	}

	t.Run("Sync Example Provider Resources", func(t *testing.T) {
		// Load specs from testdata directory
		err := runSync(t, map[string]string{
			"books/books.yaml": `version: rudder/v0.1
kind: books
metadata:
  name: my_books
spec:
  books:
    - id: "lotr"
      name: The Lord of the Rings
      author: "#/writer/common/tolkien"
    - id: "hobbit"
      name: "The Hobbit (with wrong author)"
      author: "#/writer/common/orwell"
    - id: "1984"
      name: "1984"
      author: "#/writer/common/orwell"
`,
			"writer/tolkien.yaml": `version: rudder/v0.1
kind: writer
metadata:
  name: common
spec:
  id: tolkien
  name: J.R.R. Tolkien
`,
			"writer/orwell.yaml": `version: rudder/v0.1
kind: writer
metadata:
  name: common
spec:
  id: orwell
  name: George Orwell
`,
		})
		require.NoError(t, err, "Failed to sync project specs")

		verifyContents(t, b.GetAllBooks, []*backend.RemoteBook{
			{ID: "remote-book-lotr", ExternalID: "lotr", Name: "The Lord of the Rings", AuthorID: "remote-writer-tolkien"},
			{ID: "remote-book-hobbit", ExternalID: "hobbit", Name: "The Hobbit (with wrong author)", AuthorID: "remote-writer-orwell"},
			{ID: "remote-book-1984", ExternalID: "1984", Name: "1984", AuthorID: "remote-writer-orwell"},
		})

		verifyContents(t, b.GetAllWriters, []*backend.RemoteWriter{
			{ID: "remote-writer-tolkien", ExternalID: "tolkien", Name: "J.R.R. Tolkien"},
			{ID: "remote-writer-orwell", ExternalID: "orwell", Name: "George Orwell"},
		})
	})

	t.Run("Update Example Provider Resources", func(t *testing.T) {
		// Load specs from testdata directory
		err := runSync(t, map[string]string{
			"books/books.yaml": `version: rudder/v0.1
kind: books
metadata:
  name: my_books
spec:
  books:
    - id: "lotr"
      name: The Lord of the Rings
      author: "#/writer/common/tolkien"
    - id: "hobbit"
      name: "The Hobbit"
      author: "#/writer/common/tolkien"
`,
			"writer/tolkien.yaml": `version: rudder/v0.1
kind: writer
metadata:
  name: common
spec:
  id: tolkien
  name: J.R.R. Tolkien
`,
		})
		require.NoError(t, err, "Failed to sync project specs")

		verifyContents(t, b.GetAllBooks, []*backend.RemoteBook{
			{ID: "remote-book-lotr", ExternalID: "lotr", Name: "The Lord of the Rings", AuthorID: "remote-writer-tolkien"},
			{ID: "remote-book-hobbit", ExternalID: "hobbit", Name: "The Hobbit", AuthorID: "remote-writer-tolkien"},
		})

		verifyContents(t, b.GetAllWriters, []*backend.RemoteWriter{
			{ID: "remote-writer-tolkien", ExternalID: "tolkien", Name: "J.R.R. Tolkien"},
		})
	})

}

type mockLoader struct {
	specs map[string]string
}

func verifyContents[T any](t *testing.T, getAll func() []T, expectedContents []T) {
	contents := getAll()
	assert.Len(t, contents, len(expectedContents), "Unexpected number of contents in backend")
	assert.ElementsMatch(t, expectedContents, contents, "Backend contents do not match expected")
}

func (m *mockLoader) Load(_ string) (map[string]*specs.Spec, error) {
	s := make(map[string]*specs.Spec, len(m.specs))
	for p, specStr := range m.specs {
		spec, err := specs.New([]byte(specStr))
		if err != nil {
			return nil, fmt.Errorf("parsing spec '%s': %w", p, err)
		}
		s[p] = spec
	}
	return s, nil
}
