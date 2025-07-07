package ui

import (
	"fmt"
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

// Ruler prints a horizontal line across the terminal.
func Ruler() {
	width := GetTerminalWidth()
	fmt.Println(strings.Repeat("─", width))
}
