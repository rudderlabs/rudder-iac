package ui

import "fmt"

func PrintSuccess(message string) {
	fmt.Printf("%s %s\n", Color("✓", Green), message)
}

func PrintFailure(message error) {
	fmt.Printf("%s %s\n", Color("x", Red), message)
}
