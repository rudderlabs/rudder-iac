package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

func PrintError(err error) {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(Red)).
		Bold(true).
		Render

	fmt.Fprintf(os.Stderr, "%s%s\n", style("Error: "), err.Error())
}

func PrintWarning(message string) {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(Yellow)).
		Bold(true).
		Render

	fmt.Fprintf(os.Stderr, "%s%s\n", style("Warning: "), message)
}
