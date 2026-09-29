package generator

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/kotlin"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/swift"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/typescript"
)

var platforms = map[string]core.Generator{
	"kotlin":     &kotlin.Generator{},
	"swift":      &swift.Generator{},
	"typescript": &typescript.Generator{},
}

func GeneratorForPlatform(platform string) (core.Generator, error) {
	if generator, ok := platforms[platform]; ok {
		return generator, nil
	}
	return nil, fmt.Errorf("unsupported platform: %s", platform)
}
