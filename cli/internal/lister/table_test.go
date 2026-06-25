package lister

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintTableWithDetails_WhenNoResources_PrintsPlainMessage(t *testing.T) {
	var buf bytes.Buffer
	ui.SetWriter(&buf)
	t.Cleanup(ui.RestoreWriter)

	printErr := printTableWithDetails(nil, nil)
	require.NoError(t, printErr)

	require.Equal(t, "No resources found\n", buf.String())
}

func TestModel_Init(t *testing.T) {
	t.Parallel()

	m := model{}
	assert.Nil(t, m.Init())
}

func TestModel_KeyMapHelp(t *testing.T) {
	t.Parallel()

	shortHelp := keys.ShortHelp()
	fullHelp := keys.FullHelp()

	assert.Len(t, shortHelp, 3)
	assert.Len(t, fullHelp, 2)
	assert.Len(t, fullHelp[0], 2)
	assert.Len(t, fullHelp[1], 1)
}

func TestModel_View_WithNoResources(t *testing.T) {
	t.Parallel()

	m := model{
		table:     table.New(),
		help:      help.New(),
		keys:      keys,
		resources: nil,
		width:     120,
	}

	view := m.View()
	assert.Contains(t, view, noResourcesFoundMsg)
}

func TestModel_View_WithResources(t *testing.T) {
	t.Parallel()

	rs := []resources.ResourceData{
		{"id": "res-1", "name": "Resource One"},
	}

	rows := []table.Row{{"1", "res-1", "Resource One"}}
	tbl := table.New(
		table.WithColumns([]table.Column{
			{Title: "#", Width: 4},
			{Title: "ID", Width: 27},
			{Title: "Name", Width: 30},
		}),
		table.WithRows(rows),
	)

	m := model{
		table:     tbl,
		help:      help.New(),
		keys:      keys,
		resources: rs,
		width:     120,
	}

	view := m.View()
	assert.Contains(t, view, "Resource One")
	assert.Contains(t, view, "res-1")
}

func TestModel_Update_WindowSize(t *testing.T) {
	t.Parallel()

	m := model{
		table: table.New(),
		help:  help.New(),
		keys:  keys,
	}

	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	updatedModel := updated.(model)

	assert.Nil(t, cmd)
	assert.Equal(t, 100, updatedModel.width)
	assert.Equal(t, 40, updatedModel.height)
	assert.Equal(t, 100, updatedModel.help.Width)
}

func TestModel_Update_Quit(t *testing.T) {
	t.Parallel()

	m := model{
		table: table.New(),
		help:  help.New(),
		keys:  keys,
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.NotNil(t, cmd)
}

func TestKeyMap_ShortHelpBindings(t *testing.T) {
	t.Parallel()

	km := keyMap{
		Up:   key.NewBinding(key.WithKeys("up")),
		Down: key.NewBinding(key.WithKeys("down")),
		Quit: key.NewBinding(key.WithKeys("esc")),
	}

	bindings := km.ShortHelp()
	assert.Len(t, bindings, 3)
}
