package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

const defaultTerminalWidth = 80

func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

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

// Separator returns a string of the given character repeated to the given width.
func Separator(char string, width int) string {
	if width <= 0 {
		return ""
	}
	return strings.Repeat(char, width)
}

// PadColumns formats two strings with padding between them so that rightText
// starts at the given column position. Uses ANSI-aware width measurement.
func PadColumns(leftText, rightText string, rightCol int) string {
	padding := max(rightCol-lipgloss.Width(leftText), 1)
	return fmt.Sprintf("%s%s%s", leftText, strings.Repeat(" ", padding), rightText)
}

// SectionHeader returns a bold title followed by a separator line.
func SectionHeader(title, separatorChar string, width int) string {
	return fmt.Sprintf("%s\n%s", Bold(title), Separator(separatorChar, width))
}
