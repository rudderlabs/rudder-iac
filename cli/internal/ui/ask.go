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
