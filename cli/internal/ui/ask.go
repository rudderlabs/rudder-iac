package ui

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

// AskSecret asks the user a question and reads the response as a secret.
// The response is trimmed of leading and trailing whitespace.
func AskSecret(question string) (string, error) {
	response := ""
	prompt := &survey.Password{
		Message: question,
	}

	if err := survey.AskOne(prompt, &response); err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	response = strings.TrimSpace(response)

	return response, nil
}

// Ask asks the user a free-text question and returns their trimmed response.
// A non-empty def is offered as the default, so pressing enter accepts it.
func Ask(question, def string) (string, error) {
	response := ""
	prompt := &survey.Input{
		Message: question,
		Default: def,
	}

	if err := survey.AskOne(prompt, &response); err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// Select asks the user to pick one of options and returns the chosen value.
// It returns an error rather than a silent zero value when options is empty,
// because an empty option set means the caller filtered everything out.
func Select(question string, options []string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no options to choose from for %q", question)
	}

	response := ""
	prompt := &survey.Select{
		Message: question,
		Options: options,
	}

	if err := survey.AskOne(prompt, &response); err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	return response, nil
}

// Confirm asks the user a yes/no question and returns their response as a boolean.
func Confirm(question string) (bool, error) {
	response := false
	prompt := &survey.Confirm{
		Message: question,
	}

	if err := survey.AskOne(prompt, &response); err != nil {
		return false, fmt.Errorf("error reading response: %w", err)
	}

	return response, nil
}
