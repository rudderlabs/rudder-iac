package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchema_JSONMarshaling(t *testing.T) {
	t.Parallel()

	t.Run("MarshalToJSON", func(t *testing.T) {
		t.Parallel()

		schema := Schema{
			UID:             "test-uid-123",
			WriteKey:        "test-write-key",
			EventType:       "track",
			EventIdentifier: "product_viewed",
			Schema: map[string]interface{}{
				"userId":      "string",
				"anonymousId": "string",
				"event":       "string",
				"properties": map[string]interface{}{
					"product_id":   "string",
					"product_name": "string",
					"price":        "number",
				},
			},
			CreatedAt: "2024-01-10T10:08:15.407491Z",
			LastSeen:  "2024-03-25T18:49:31.870834Z",
			Count:     42,
		}

		jsonData, err := json.Marshal(schema)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		// Verify that the JSON contains expected fields
		jsonString := string(jsonData)
		assert.Contains(t, jsonString, "test-uid-123")
		assert.Contains(t, jsonString, "test-write-key")
		assert.Contains(t, jsonString, "track")
		assert.Contains(t, jsonString, "product_viewed")
		assert.Contains(t, jsonString, "2024-01-10T10:08:15.407491Z")
		assert.Contains(t, jsonString, "42")
	})

	t.Run("UnmarshalFromJSON", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"uid": "test-uid-456",
			"writeKey": "test-write-key-2",
			"eventType": "identify",
			"eventIdentifier": "user_signup",
			"schema": {
				"userId": "string",
				"traits": {
					"email": "string",
					"firstName": "string",
					"lastName": "string"
				}
			},
			"createdAt": "2024-02-15T14:30:00.000000Z",
			"lastSeen": "2024-03-30T12:45:30.123456Z",
			"count": 15
		}`

		var schema Schema
		err := json.Unmarshal([]byte(jsonData), &schema)
		require.NoError(t, err)

		assert.Equal(t, "test-uid-456", schema.UID)
		assert.Equal(t, "test-write-key-2", schema.WriteKey)
		assert.Equal(t, "identify", schema.EventType)
		assert.Equal(t, "user_signup", schema.EventIdentifier)
		assert.Equal(t, "2024-02-15T14:30:00.000000Z", schema.CreatedAt)
		assert.Equal(t, "2024-03-30T12:45:30.123456Z", schema.LastSeen)
		assert.Equal(t, 15, schema.Count)

		// Verify nested schema structure
		require.NotNil(t, schema.Schema)
		assert.Equal(t, "string", schema.Schema["userId"])

		traits, ok := schema.Schema["traits"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "string", traits["email"])
		assert.Equal(t, "string", traits["firstName"])
		assert.Equal(t, "string", traits["lastName"])
	})

	t.Run("RoundTripMarshaling", func(t *testing.T) {
		t.Parallel()

		original := Schema{
			UID:             "round-trip-test",
			WriteKey:        "test-key",
			EventType:       "page",
			EventIdentifier: "home_page_viewed",
			Schema: map[string]interface{}{
				"anonymousId": "string",
				"context": map[string]interface{}{
					"ip":        "string",
					"userAgent": "string",
					"page": map[string]interface{}{
						"title": "string",
						"url":   "string",
					},
				},
				"properties": map[string]interface{}{
					"category": "string",
					"referrer": "string",
				},
			},
			CreatedAt: "2024-01-01T00:00:00.000000Z",
			LastSeen:  "2024-12-31T23:59:59.999999Z",
			Count:     100,
		}

		// Marshal to JSON
		jsonData, err := json.Marshal(original)
		require.NoError(t, err)

		// Unmarshal back to struct
		var restored Schema
		err = json.Unmarshal(jsonData, &restored)
		require.NoError(t, err)

		// Compare all fields
		assert.Equal(t, original.UID, restored.UID)
		assert.Equal(t, original.WriteKey, restored.WriteKey)
		assert.Equal(t, original.EventType, restored.EventType)
		assert.Equal(t, original.EventIdentifier, restored.EventIdentifier)
		assert.Equal(t, original.CreatedAt, restored.CreatedAt)
		assert.Equal(t, original.LastSeen, restored.LastSeen)
		assert.Equal(t, original.Count, restored.Count)

		// Compare schema maps (deep comparison)
		assert.Equal(t, len(original.Schema), len(restored.Schema))
		for key, value := range original.Schema {
			assert.Equal(t, value, restored.Schema[key])
		}
	})
}

func TestSchema_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("EmptySchema", func(t *testing.T) {
		t.Parallel()

		schema := Schema{}
		jsonData, err := json.Marshal(schema)
		require.NoError(t, err)

		var restored Schema
		err = json.Unmarshal(jsonData, &restored)
		require.NoError(t, err)

		assert.Equal(t, schema, restored)
	})

	t.Run("NilSchemaMap", func(t *testing.T) {
		t.Parallel()

		schema := Schema{
			UID:             "nil-schema-test",
			WriteKey:        "test-key",
			EventType:       "track",
			EventIdentifier: "test_event",
			Schema:          nil,
			Count:           0,
		}

		jsonData, err := json.Marshal(schema)
		require.NoError(t, err)

		var restored Schema
		err = json.Unmarshal(jsonData, &restored)
		require.NoError(t, err)

		assert.Equal(t, schema.UID, restored.UID)
		assert.Equal(t, schema.WriteKey, restored.WriteKey)
		assert.Equal(t, schema.EventType, restored.EventType)
		assert.Equal(t, schema.EventIdentifier, restored.EventIdentifier)
		assert.Equal(t, schema.Count, restored.Count)
		assert.Nil(t, restored.Schema)
	})

	t.Run("ComplexNestedSchema", func(t *testing.T) {
		t.Parallel()

		schema := Schema{
			UID:             "complex-test",
			WriteKey:        "complex-key",
			EventType:       "track",
			EventIdentifier: "complex_event",
			Schema: map[string]interface{}{
				"simpleString": "string",
				"simpleNumber": "number",
				"simpleBool":   "boolean",
				"arrayField":   []interface{}{"string", "number"},
				"nestedObject": map[string]interface{}{
					"level1": map[string]interface{}{
						"level2": map[string]interface{}{
							"level3": "deep_value",
						},
					},
				},
				"arrayOfObjects": []interface{}{
					map[string]interface{}{
						"name":  "string",
						"value": "number",
					},
					map[string]interface{}{
						"type": "boolean",
						"data": []interface{}{"string"},
					},
				},
			},
			Count: 999,
		}

		jsonData, err := json.Marshal(schema)
		require.NoError(t, err)

		var restored Schema
		err = json.Unmarshal(jsonData, &restored)
		require.NoError(t, err)

		assert.Equal(t, schema.UID, restored.UID)
		assert.Equal(t, schema.WriteKey, restored.WriteKey)
		assert.Equal(t, schema.EventType, restored.EventType)
		assert.Equal(t, schema.EventIdentifier, restored.EventIdentifier)
		assert.Equal(t, schema.Count, restored.Count)

		// Verify complex nested structure
		assert.Equal(t, "string", restored.Schema["simpleString"])
		assert.Equal(t, "number", restored.Schema["simpleNumber"])
		assert.Equal(t, "boolean", restored.Schema["simpleBool"])

		arrayField, ok := restored.Schema["arrayField"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, "string", arrayField[0])
		assert.Equal(t, "number", arrayField[1])
	})
}

func TestSchemasFile_JSONMarshaling(t *testing.T) {
	t.Parallel()

	t.Run("MarshalSchemasFile", func(t *testing.T) {
		t.Parallel()

		schemasFile := SchemasFile{
			Schemas: []Schema{
				{
					UID:             "schema-1",
					WriteKey:        "key-1",
					EventType:       "track",
					EventIdentifier: "event_1",
					Schema: map[string]interface{}{
						"userId": "string",
					},
					Count: 10,
				},
				{
					UID:             "schema-2",
					WriteKey:        "key-2",
					EventType:       "identify",
					EventIdentifier: "event_2",
					Schema: map[string]interface{}{
						"traits": map[string]interface{}{
							"email": "string",
						},
					},
					Count: 20,
				},
			},
		}

		jsonData, err := json.Marshal(schemasFile)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		jsonString := string(jsonData)
		assert.Contains(t, jsonString, "schema-1")
		assert.Contains(t, jsonString, "schema-2")
		assert.Contains(t, jsonString, "key-1")
		assert.Contains(t, jsonString, "key-2")
	})

	t.Run("UnmarshalSchemasFile", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"schemas": [
				{
					"uid": "test-schema-1",
					"writeKey": "test-key-1",
					"eventType": "track",
					"eventIdentifier": "purchase",
					"schema": {
						"userId": "string",
						"properties": {
							"amount": "number",
							"currency": "string"
						}
					},
					"createdAt": "2024-01-01T00:00:00Z",
					"lastSeen": "2024-01-02T00:00:00Z",
					"count": 5
				},
				{
					"uid": "test-schema-2",
					"writeKey": "test-key-2",
					"eventType": "page",
					"eventIdentifier": "home",
					"schema": {
						"anonymousId": "string",
						"properties": {
							"title": "string",
							"url": "string"
						}
					},
					"createdAt": "2024-01-03T00:00:00Z",
					"lastSeen": "2024-01-04T00:00:00Z",
					"count": 15
				}
			]
		}`

		var schemasFile SchemasFile
		err := json.Unmarshal([]byte(jsonData), &schemasFile)
		require.NoError(t, err)

		assert.Len(t, schemasFile.Schemas, 2)

		// Check first schema
		schema1 := schemasFile.Schemas[0]
		assert.Equal(t, "test-schema-1", schema1.UID)
		assert.Equal(t, "test-key-1", schema1.WriteKey)
		assert.Equal(t, "track", schema1.EventType)
		assert.Equal(t, "purchase", schema1.EventIdentifier)
		assert.Equal(t, 5, schema1.Count)

		// Check second schema
		schema2 := schemasFile.Schemas[1]
		assert.Equal(t, "test-schema-2", schema2.UID)
		assert.Equal(t, "test-key-2", schema2.WriteKey)
		assert.Equal(t, "page", schema2.EventType)
		assert.Equal(t, "home", schema2.EventIdentifier)
		assert.Equal(t, 15, schema2.Count)
	})

	t.Run("EmptySchemasFile", func(t *testing.T) {
		t.Parallel()

		schemasFile := SchemasFile{
			Schemas: []Schema{},
		}

		jsonData, err := json.Marshal(schemasFile)
		require.NoError(t, err)

		var restored SchemasFile
		err = json.Unmarshal(jsonData, &restored)
		require.NoError(t, err)

		assert.Equal(t, 0, len(restored.Schemas))
	})

	t.Run("NilSchemasSlice", func(t *testing.T) {
		t.Parallel()

		schemasFile := SchemasFile{
			Schemas: nil,
		}

		jsonData, err := json.Marshal(schemasFile)
		require.NoError(t, err)

		var restored SchemasFile
		err = json.Unmarshal(jsonData, &restored)
		require.NoError(t, err)

		assert.Nil(t, restored.Schemas)
	})
}
