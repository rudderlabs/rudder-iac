package iac

import (
	"fmt"
	"path/filepath"
)

type PulumiConfig struct {
	Project     string
	Stack       string
	PulumiHome  string
	ToolVersion string
}

func (conf PulumiConfig) GetQualifiedStack() string {
	return fmt.Sprintf("organization/%s/%s", conf.Project, conf.Stack)
}

func (conf PulumiConfig) GetProject() string {
	return conf.Project
}

func (conf PulumiConfig) GetWorkDir() string {
	return filepath.Join(conf.PulumiHome, "work")
}

func (conf PulumiConfig) GetHomeDir() string {
	return conf.PulumiHome
}

func (conf PulumiConfig) GetPulumiVersion() string {
	return conf.ToolVersion
}
