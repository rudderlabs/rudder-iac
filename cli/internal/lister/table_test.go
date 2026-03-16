package lister

import (
	"testing"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func testColumns() []table.Column {
	return []table.Column{
		{Title: "#", Width: 4},
		{Title: "ID", Width: 27},
		{Title: "Name", Width: 30},
	}
}

func TestModel_WindowSizeMsg_SetsTableHeight(t *testing.T) {
	rows := make([]table.Row, 100)
	for i := range rows {
		rows[i] = table.Row{"1", "id", "name"}
	}

	m := model{
		table: table.New(
			table.WithColumns(testColumns()),
			table.WithRows(rows),
			table.WithHeight(20),
		),
		help: help.New(),
		keys: keys,
	}

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(msg)
	updatedModel := updated.(model)

	// table.Height() returns viewport height (SetHeight value minus 1 for header)
	setHeight := 40 - 1 - 1 // height - helpHeight - margin
	assert.Equal(t, setHeight-1, updatedModel.table.Height())
	assert.Less(t, updatedModel.table.Height(), len(rows))
}

func TestModel_WindowSizeMsg_EnforcesMinimumHeight(t *testing.T) {
	m := model{
		table: table.New(
			table.WithColumns(testColumns()),
			table.WithHeight(20),
		),
		help: help.New(),
		keys: keys,
	}

	msg := tea.WindowSizeMsg{Width: 80, Height: 3}
	updated, _ := m.Update(msg)
	updatedModel := updated.(model)

	// min height of 3 is passed to SetHeight; Height() returns 3-1=2
	assert.Equal(t, 2, updatedModel.table.Height())
}
