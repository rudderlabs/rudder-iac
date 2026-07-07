package handler

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
)

// SecretField describes a secret-bearing field declared by a handler resource or state type.
type SecretField struct {
	JSONName string
}

type secretFieldConfigurator interface {
	SetSecretFields([]SecretField)
}

var (
	secretStringType    = reflect.TypeOf(secret.String{})
	secretStringPtrType = reflect.TypeOf((*secret.String)(nil))
)

func declaredSecretFields[Res any, State any]() []SecretField {
	fields := map[string]SecretField{}
	collectDeclaredSecretFields(typeOf[Res](), fields, map[reflect.Type]bool{})
	collectDeclaredSecretFields(typeOf[State](), fields, map[reflect.Type]bool{})

	jsonNames := make([]string, 0, len(fields))
	for jsonName := range fields {
		jsonNames = append(jsonNames, jsonName)
	}
	sort.Strings(jsonNames)

	result := make([]SecretField, 0, len(jsonNames))
	for _, jsonName := range jsonNames {
		result = append(result, fields[jsonName])
	}
	return result
}

func typeOf[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func collectDeclaredSecretFields(t reflect.Type, fields map[string]SecretField, seen map[reflect.Type]bool) {
	structType, ok := secretSearchStructType(t)
	if !ok || seen[structType] {
		return
	}
	seen[structType] = true

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}
		if registerSecretField(field, fields) {
			continue
		}
		collectDeclaredSecretFields(field.Type, fields, seen)
	}
}

func secretSearchStructType(t reflect.Type) (reflect.Type, bool) {
	for t != nil {
		if isSecretStringType(t) {
			return nil, false
		}
		switch t.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Array:
			t = t.Elem()
		case reflect.Struct:
			return t, true
		default:
			return nil, false
		}
	}
	return nil, false
}

func registerSecretField(field reflect.StructField, fields map[string]SecretField) bool {
	if field.Tag.Get("secret") != "true" || !isSecretStringType(field.Type) {
		return false
	}

	jsonName := jsonFieldName(field)
	if jsonName != "" {
		fields[jsonName] = SecretField{JSONName: jsonName}
	}
	return true
}

func scrubDeclaredSecrets(target any) error {
	if target == nil {
		return nil
	}
	return scrubSecretValue(reflect.ValueOf(target))
}

func scrubSecretValue(v reflect.Value) error {
	value, ok := indirectValue(v)
	if !ok {
		return nil
	}

	switch value.Kind() {
	case reflect.Struct:
		if value.Type() == secretStringType {
			return nil
		}
		return scrubSecretStruct(value)
	case reflect.Slice, reflect.Array:
		return scrubSecretSequence(value)
	}
	return nil
}

func indirectValue(v reflect.Value) (reflect.Value, bool) {
	if !v.IsValid() {
		return reflect.Value{}, false
	}
	for v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}, false
		}
		v = v.Elem()
	}
	return v, true
}

func scrubSecretSequence(v reflect.Value) error {
	for i := 0; i < v.Len(); i++ {
		if err := scrubSecretValue(v.Index(i)); err != nil {
			return err
		}
	}
	return nil
}

func scrubSecretStruct(v reflect.Value) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldValue := v.Field(i)
		if field.Tag.Get("secret") == "true" && isSecretStringType(field.Type) {
			if err := setUnknownSecret(field.Name, fieldValue); err != nil {
				return err
			}
			continue
		}
		if err := scrubSecretValue(fieldValue); err != nil {
			return err
		}
	}
	return nil
}

func setUnknownSecret(fieldName string, v reflect.Value) error {
	if !v.CanSet() {
		return fmt.Errorf("setting secret field %s: field is not settable", fieldName)
	}

	unknown := secret.NewUnknown()
	switch v.Type() {
	case secretStringType:
		v.Set(reflect.ValueOf(unknown))
	case secretStringPtrType:
		v.Set(reflect.ValueOf(&unknown))
	}
	return nil
}

func isSecretStringType(t reflect.Type) bool {
	return t == secretStringType || t == secretStringPtrType
}

func jsonFieldName(field reflect.StructField) string {
	name := strings.Split(field.Tag.Get("json"), ",")[0]
	if name == "-" {
		return ""
	}
	if name != "" {
		return name
	}
	return field.Name
}
