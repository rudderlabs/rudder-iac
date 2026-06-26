package main

import "github.com/rudderlabs/rudder-iac/cli/internal/apicmd"

var version string = "0.0.0"

func main() {
	apicmd.SetVersion(version)
	apicmd.Execute()
}
