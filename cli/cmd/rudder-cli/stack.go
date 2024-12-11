package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
)

func main1() {

	// ws, err := auto.NewLocalWorkspace(context.Background(), auto.WorkDir("."))
	// if err != nil {
	// 	log.Fatalf("creating local workspace: %w", err)
	// }

	// fmt.Printf("workspace: %v", ws)

	os.Setenv("PULUMI_CONFIG_PASSPHRASE", "")
	s, err := auto.UpsertStackInlineSource(context.Background(), "organization/myproj/test", "myproj", nil, auto.WorkDir("."), auto.Project(workspace.Project{
		Name:    "myproj",
		Runtime: workspace.NewProjectRuntimeInfo("go", nil),
	}))

	if err != nil {
		log.Fatalf("creating stack: %s", err)
	}

	fmt.Printf("stack: %v", s)
}
