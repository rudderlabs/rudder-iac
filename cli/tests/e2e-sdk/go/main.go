// Command probe sends one track with the Go SDK, gzip on by default.
// Usage: go run . DATA_PLANE_URL
package main

import (
	"fmt"
	"os"

	analytics "github.com/rudderlabs/analytics-go/v4"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run . DATA_PLANE_URL")
		os.Exit(2)
	}
	client, err := analytics.NewWithConfig("go", analytics.Config{DataPlaneUrl: os.Args[1]})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = client.Enqueue(analytics.Track{
		UserId:     "u1",
		Event:      "SDK Probe",
		Properties: analytics.NewProperties().Set("sdk", "go").Set("n", 1),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// Close flushes the queue.
	if err := client.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("go sent")
}
