package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

func PrintTable(columns []table.Column, rows []table.Row) {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(len(rows)+1), // +1 for header
		table.WithFocused(false),      // Disable focus/selection for static display
	)

	// Style the table
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Bold(true)
	// Remove selected row styling for static display
	s.Selected = lipgloss.NewStyle()
	t.SetStyles(s)

	// Print the table
	fmt.Println(t.View())
}
