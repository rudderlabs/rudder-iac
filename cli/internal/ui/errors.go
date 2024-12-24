package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func ShowError(err error) {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(Red)).
		Bold(true).
		Render

	fmt.Println(style("Error: ") + err.Error())
}
