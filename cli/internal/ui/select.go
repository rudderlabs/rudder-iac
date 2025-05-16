package ui

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
)

// Option represents a selectable option with an ID and display value
type Option struct {
	// ID is the unique identifier for the option that will be returned when selected
	ID string
	// Display is the text shown to the user in the selection UI
	Display string
}

// Select presents a list of options to the user and returns the ID of their selection.
// options must be non-empty. The returned string will be the ID of the selected option.
func Select(message string, options []Option) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no options provided for selection")
	}

	// Create display options and ID mapping
	displays := make([]string, len(options))
	idMap := make(map[string]string)
	for i, opt := range options {
		displays[i] = opt.Display
		idMap[opt.Display] = opt.ID
	}

	var selected string
	prompt := &survey.Select{
		Message: message,
		Options: displays,
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		return "", fmt.Errorf("error reading selection: %w", err)
	}

	return idMap[selected], nil
}

// MultiSelect presents a list of options to the user and returns the IDs of their selections.
// options must be non-empty. The returned strings will be the IDs of the selected options.
func MultiSelect(message string, options []Option) ([]string, error) {
	if len(options) == 0 {
		return nil, fmt.Errorf("no options provided for selection")
	}

	// Create display options and ID mapping
	displays := make([]string, len(options))
	idMap := make(map[string]string)
	for i, opt := range options {
		displays[i] = opt.Display
		idMap[opt.Display] = opt.ID
	}

	var selectedDisplays []string
	prompt := &survey.MultiSelect{
		Message: message,
		Options: displays,
	}

	if err := survey.AskOne(prompt, &selectedDisplays); err != nil {
		return nil, fmt.Errorf("error reading selections: %w", err)
	}

	// Convert selected displays back to IDs
	selectedIDs := make([]string, len(selectedDisplays))
	for i, display := range selectedDisplays {
		selectedIDs[i] = idMap[display]
	}

	return selectedIDs, nil
}
