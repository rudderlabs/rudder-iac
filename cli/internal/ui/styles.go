package ui

import "github.com/charmbracelet/lipgloss"

func Bold(s string) string {
	return lipgloss.NewStyle().Bold(true).Render(s)
}

func Color(s string, color string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(s)
}

func GreyedOut(s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#999999")).Render(s)
}
