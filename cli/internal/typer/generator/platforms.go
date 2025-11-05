package generator

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/kotlin"
)

var platforms = map[string]core.Generator{
	"kotlin": &kotlin.Generator{},
}

func GeneratorForPlatform(platform string) (core.Generator, error) {
	if generator, ok := platforms[platform]; ok {
		return generator, nil
	}
	return nil, fmt.Errorf("unsupported platform: %s", platform)
}
