package main

import "github.com/rudderlabs/rudder-iac/cli/internal/cmd"

var version string = "0.0.0"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
