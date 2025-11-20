package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// PrintError prints the given error message to stderr in a styled format.
func PrintError(err error) {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorRed)).
		Bold(true).
		Render

	fmt.Fprintf(uiErrWriter, "%s %s\n", style("Error:"), err.Error())
}

func Warning(message string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorYellow)).
		Bold(true).
		Render

	return fmt.Sprintf("%s %s", style("Warning:"), message)
}

func PrintWarning(message string) {
	fmt.Println(Warning(message))
}

// Failure returns a failure message string in a styled format, including a red "x" symbol.
func Failure(message string) string {
	return fmt.Sprintf("%s %s", Color("x", ColorRed), message)
}

// PrintFailure prints a failure message to stdout in a styled format, including a red "x" symbol.
func PrintFailure(message string) {
	fmt.Fprintln(uiWriter, Failure(message))
}

// Success returns a success message string in a styled format, including a green checkmark symbol.
func Success(message string) string {
	return fmt.Sprintf("%s %s", Color("✔", ColorGreen), message)
}

// PrintSuccess prints a success message to stdout in a styled format, including a green checkmark symbol.
func PrintSuccess(message string) {
	fmt.Fprintln(uiWriter, Success(message))
}
