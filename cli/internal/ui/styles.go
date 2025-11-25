package ui

import "github.com/charmbracelet/lipgloss"

// Bold returns the input string styled in bold.
func Bold(s string) string {
	return lipgloss.NewStyle().Bold(true).Render(s)
}

const (
	ColorRed    string = "9"
	ColorGreen  string = "10"
	ColorYellow string = "11"
	ColorWhite  string = "15"
	ColorBlue   string = "12"
	ColorGrey   string = "#999999"
)

// Color returns the input string styled with the specified color,
// supported by lipgloss. This can be either an ANSI color (https://en.wikipedia.org/wiki/ANSI_escape_code#Colors)
// or an RGBA hex code.
func Color(s string, color string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(s)
}

// GreyedOut returns the input string styled in a greyed-out color.
func GreyedOut(s string) string {
	return Color(s, ColorGrey)
}
