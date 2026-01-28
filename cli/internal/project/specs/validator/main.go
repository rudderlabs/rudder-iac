package main

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Spec struct {
	Version  string         `yaml:"version" validate:"required"`
	Kind     string         `yaml:"kind" validate:"required"`
	Metadata map[string]any `yaml:"metadata" validate:"required"`
	Spec     map[string]any `yaml:"spec" validate:"required"`
	Inners   []InnerSpec    `yaml:"inners" validate:"dive"`
}

type InnerSpec struct {
	Name    string `yaml:"name" validate:"required"`
	Surname string `yaml:"surname" validate:"required"`
}

// arrayIndexRegex matches array indices like [0], [1], etc.
var arrayIndexRegex = regexp.MustCompile(`\[(\d+)\]`)

// NamespaceToJSONPointer converts validator's StructNamespace to JSON Pointer format.
// When used with RegisterTagNameFunc for yaml tags, the namespace already contains
// yaml field names, so we just need to format it as a JSON pointer.
//
// Example: "Spec.inners[1].surname" → "/inners/1/surname"
func NamespaceToJSONPointer(namespace string) string {
	// Remove root struct name (everything before first dot)
	if idx := strings.Index(namespace, "."); idx != -1 {
		namespace = namespace[idx+1:]
	}

	// Convert array indices: [N] → /N
	namespace = arrayIndexRegex.ReplaceAllString(namespace, "/$1")

	// Convert dots to slashes
	namespace = strings.ReplaceAll(namespace, ".", "/")

	return "/" + namespace
}

func main() {
	s := Spec{
		Version:  "rudder/v1",
		Kind:     "",
		Metadata: nil,
		Spec:     map[string]any{"properties": []any{}},
		Inners: []InnerSpec{
			{
				Name:    "John",
				Surname: "Doe",
			},
			{
				Name: "Jane",
			},
		},
	}

	v := validator.New()

	// Register tag name function to use yaml tag names instead of Go field names.
	// This makes StructNamespace() return "Spec.inners[1].surname" instead of "Spec.Inners[1].Surname"
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("yaml"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	if err := v.Struct(s); err != nil {
		var vErrs validator.ValidationErrors
		errors.As(err, &vErrs)

		for _, vErr := range vErrs {
			// StructNamespace() returns Go field names (e.g., "Spec.Inners[1].Surname")
			// Namespace() returns registered tag names (e.g., "Spec.inners[1].surname")
			structNamespace := vErr.StructNamespace()
			tagNamespace := vErr.Namespace()

			// Convert tag namespace to JSON Pointer for validation engine
			jsonPointer := NamespaceToJSONPointer(tagNamespace)

			fmt.Printf("StructNamespace: %s\n", structNamespace)
			fmt.Printf("Namespace (tag): %s\n", tagNamespace)
			fmt.Printf("JSON Pointer:    %s\n", jsonPointer)
			fmt.Println("---")
		}
	}
}
