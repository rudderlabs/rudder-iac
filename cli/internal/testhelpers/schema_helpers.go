package testhelpers

// CreateComplexSchema creates a comprehensive test schema with nested structures
func CreateComplexSchema(uid, writeKey, eventIdentifier string) map[string]interface{} {
	return map[string]interface{}{
		"uid":             uid,
		"writeKey":        writeKey,
		"eventType":       "track",
		"eventIdentifier": eventIdentifier,
		"schema": map[string]interface{}{
			"event":       "string",
			"userId":      "string",
			"anonymousId": "string",
			"properties": map[string]interface{}{
				"simple_prop":   "string",
				"number_prop":   "number",
				"nested_object": map[string]interface{}{"field1": "string", "field2": "number"},
				"array_prop":    []interface{}{"string", "number"},
			},
			"context": map[string]interface{}{
				"app":    map[string]interface{}{"name": "string", "version": "string"},
				"device": map[string]interface{}{"type": "string", "model": "string"},
			},
		},
	}
}

// CreateIdentifySchema creates an identify event test schema
func CreateIdentifySchema(uid, writeKey, eventIdentifier string) map[string]interface{} {
	return map[string]interface{}{
		"uid":             uid,
		"writeKey":        writeKey,
		"eventType":       "identify",
		"eventIdentifier": eventIdentifier,
		"schema": map[string]interface{}{
			"event":  "string",
			"userId": "string",
			"traits": map[string]interface{}{"email": "string", "name": "string", "age": "number"},
		},
	}
}
