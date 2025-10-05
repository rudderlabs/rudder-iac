package debug

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newCmdStacktrace() *cobra.Command {
	return &cobra.Command{
		Use:   "stacktrace",
		Short: "Display the last panic stacktrace from the log file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("error getting home directory: %w", err)
			}

			logPath := filepath.Join(homeDir, ".rudder", "cli.log")
			file, err := os.Open(logPath)
			if err != nil {
				return fmt.Errorf("error opening log file: %w", err)
			}
			defer file.Close()

			// Read the last ERROR entry
			var lastErrorMsg string
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(line, "level=ERROR") {
					lastErrorMsg = line
				}
			}

			if err := scanner.Err(); err != nil {
				return fmt.Errorf("error reading log file: %w", err)
			}

			if lastErrorMsg == "" {
				fmt.Println("No panic stacktrace found in log file")
				return nil
			}

			// Extract and format the stacktrace from the msg field
			stacktrace := extractStacktrace(lastErrorMsg)
			if stacktrace == "" {
				fmt.Println("No stacktrace found in last error entry")
				return nil
			}

			// Print formatted stacktrace
			fmt.Println("Last Panic Stacktrace:")
			fmt.Println("======================")
			fmt.Println(stacktrace)

			return nil
		},
	}
}

// extractStacktrace extracts and formats the stacktrace from a log entry
func extractStacktrace(logEntry string) string {
	// Find the msg field which contains the stacktrace
	msgStart := strings.Index(logEntry, "msg=\"")
	if msgStart == -1 {
		return ""
	}
	msgStart += 5 // Move past 'msg="'

	// Find the end of the msg field (the closing quote before the next field)
	msgEnd := strings.Index(logEntry[msgStart:], "\" ")
	if msgEnd == -1 {
		// Try to find just the closing quote at the end
		msgEnd = strings.LastIndex(logEntry[msgStart:], "\"")
		if msgEnd == -1 {
			return ""
		}
	}

	stacktrace := logEntry[msgStart : msgStart+msgEnd]

	// Replace escaped newlines and tabs with actual newlines and tabs
	stacktrace = strings.ReplaceAll(stacktrace, "\\n", "\n")
	stacktrace = strings.ReplaceAll(stacktrace, "\\t", "\t")

	return stacktrace
}
