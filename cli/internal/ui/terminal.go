package ui

import (
	"os"
	"strings"

	"golang.org/x/term"
)

const defaultTerminalWidth = 80

// GetTerminalWidth returns the current width of the terminal.
func GetTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return defaultTerminalWidth
	}
	return width
}

// Ruler returns a string representing a horizontal line across the terminal.
func Ruler() string {
	return RulerWithWidth(GetTerminalWidth())
}

// RulerWithWidth returns a string representing a horizontal line of a given width.
func RulerWithWidth(width int) string {
	if width <= 0 {
		return ""
	}

	return strings.Repeat("─", width)
}
