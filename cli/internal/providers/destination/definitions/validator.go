package definitions

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-viper/mapstructure/v2"

	prulefuncs "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func validateConfigModel(config map[string]any, configType reflect.Type, basePath string) []ConfigError {
	configType = derefType(configType)
	if configType == nil || configType.Kind() != reflect.Struct {
		return []ConfigError{{
			Path:    basePath,
			Message: "config model must be a struct",
		}}
	}

	errors := findUnknownKeys(config, configType, basePath)

	decoded := reflect.New(configType).Interface()
	if err := decodeConfig(config, decoded); err != nil {
		errors = append(errors, ConfigError{
			Path:    basePath,
			Message: err.Error(),
		})
		return errors
	}

	errors = append(errors, structValidationErrors(decoded, basePath, configType)...)
	return errors
}

func decodeConfig(config map[string]any, dest any) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           dest,
		WeaklyTypedInput: false,
	})
	if err != nil {
		return fmt.Errorf("creating config decoder: %w", err)
	}

	if err := decoder.Decode(config); err != nil {
		return fmt.Errorf("decoding config: %w", err)
	}
	return nil
}

func structValidationErrors(decoded any, basePath string, rootType reflect.Type) []ConfigError {
	validationErrors, err := rules.ValidateStructWithTagPriority(decoded, []string{"mapstructure"}, configValidateFuncs()...)
	if err != nil {
		return []ConfigError{{
			Path:    basePath,
			Message: err.Error(),
		}}
	}

	if len(validationErrors) == 0 {
		return nil
	}

	results := prulefuncs.ParseMapstructureValidationErrors(validationErrors, rootType)
	errors := make([]ConfigError, 0, len(results))
	for _, result := range results {
		path := basePath
		if result.Reference != "" {
			path = joinConfigPath(basePath, strings.TrimPrefix(result.Reference, "/"))
		}

		errors = append(errors, ConfigError{
			Path:    path,
			Message: result.Message,
		})
	}

	return errors
}
