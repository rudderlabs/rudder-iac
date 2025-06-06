package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemasResponse_JSONMarshaling(t *testing.T) {
	t.Parallel()

	t.Run("MarshalToJSON", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		response := SchemasResponse{
			Results: []Schema{
				{
					UID:             "test-uid-1",
					WriteKey:        "test-write-key-1",
					EventType:       "track",
					EventIdentifier: "product_viewed",
					Schema: map[string]interface{}{
						"userId": "string",
						"properties": map[string]interface{}{
							"product_id": "string",
						},
					},
					CreatedAt: now,
					LastSeen:  now,
					Count:     10,
				},
			},
			CurrentPage: 1,
			HasNext:     true,
		}

		jsonData, err := json.Marshal(response)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		jsonString := string(jsonData)
		assert.Contains(t, jsonString, "test-uid-1")
		assert.Contains(t, jsonString, "currentPage")
		assert.Contains(t, jsonString, "hasNext")
		assert.Contains(t, jsonString, "results")
	})

	t.Run("UnmarshalFromJSON", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"results": [
				{
					"uid": "test-uid-2",
					"writeKey": "test-write-key-2",
					"eventType": "identify",
					"eventIdentifier": "user_signup",
					"schema": {
						"userId": "string",
						"traits": {
							"email": "string"
						}
					},
					"createdAt": "2024-01-10T10:08:15.407491Z",
					"lastSeen": "2024-03-25T18:49:31.870834Z",
					"count": 5
				}
			],
			"currentPage": 2,
			"hasNext": false
		}`

		var response SchemasResponse
		err := json.Unmarshal([]byte(jsonData), &response)
		require.NoError(t, err)

		assert.Len(t, response.Results, 1)
		assert.Equal(t, 2, response.CurrentPage)
		assert.False(t, response.HasNext)

		schema := response.Results[0]
		assert.Equal(t, "test-uid-2", schema.UID)
		assert.Equal(t, "test-write-key-2", schema.WriteKey)
		assert.Equal(t, "identify", schema.EventType)
		assert.Equal(t, "user_signup", schema.EventIdentifier)
		assert.Equal(t, 5, schema.Count)
	})

	t.Run("EmptyResults", func(t *testing.T) {
		t.Parallel()

		response := SchemasResponse{
			Results:     []Schema{},
			CurrentPage: 1,
			HasNext:     false,
		}

		jsonData, err := json.Marshal(response)
		require.NoError(t, err)

		var restored SchemasResponse
		err = json.Unmarshal(jsonData, &restored)
		require.NoError(t, err)

		assert.Empty(t, restored.Results)
		assert.Equal(t, 1, restored.CurrentPage)
		assert.False(t, restored.HasNext)
	})

	t.Run("WithoutHasNext", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"results": [],
			"currentPage": 1
		}`

		var response SchemasResponse
		err := json.Unmarshal([]byte(jsonData), &response)
		require.NoError(t, err)

		assert.Equal(t, 1, response.CurrentPage)
		assert.False(t, response.HasNext) // Should default to false
	})
}

func TestSchema_JSONMarshaling(t *testing.T) {
	t.Parallel()

	t.Run("MarshalWithTimeFields", func(t *testing.T) {
		t.Parallel()

		createdAt := time.Date(2024, 1, 10, 10, 8, 15, 407491000, time.UTC)
		lastSeen := time.Date(2024, 3, 25, 18, 49, 31, 870834000, time.UTC)

		schema := Schema{
			UID:             "time-test-uid",
			WriteKey:        "time-test-key",
			EventType:       "track",
			EventIdentifier: "time_event",
			Schema: map[string]interface{}{
				"userId": "string",
				"event":  "string",
			},
			CreatedAt: createdAt,
			LastSeen:  lastSeen,
			Count:     25,
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

		// Time fields should be properly handled
		assert.True(t, schema.CreatedAt.Equal(restored.CreatedAt))
		assert.True(t, schema.LastSeen.Equal(restored.LastSeen))
	})

	t.Run("ComplexSchemaStructure", func(t *testing.T) {
		t.Parallel()

		schema := Schema{
			UID:             "complex-schema-uid",
			WriteKey:        "complex-key",
			EventType:       "track",
			EventIdentifier: "complex_event",
			Schema: map[string]interface{}{
				"userId":      "string",
				"anonymousId": "string",
				"context": map[string]interface{}{
					"app": map[string]interface{}{
						"name":    "string",
						"version": "string",
					},
					"device": map[string]interface{}{
						"type": "string",
						"id":   "string",
					},
				},
				"properties": map[string]interface{}{
					"product_id":   "string",
					"product_name": "string",
					"price":        "number",
					"categories":   []interface{}{"string"},
					"metadata": map[string]interface{}{
						"source":  "string",
						"version": "number",
					},
				},
			},
			CreatedAt: time.Now(),
			LastSeen:  time.Now(),
			Count:     100,
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

		// Verify complex nested schema structure
		assert.Equal(t, "string", restored.Schema["userId"])
		assert.Equal(t, "string", restored.Schema["anonymousId"])

		context, ok := restored.Schema["context"].(map[string]interface{})
		require.True(t, ok)

		app, ok := context["app"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "string", app["name"])
		assert.Equal(t, "string", app["version"])

		properties, ok := restored.Schema["properties"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "string", properties["product_id"])
		assert.Equal(t, "number", properties["price"])

		categories, ok := properties["categories"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, "string", categories[0])
	})

	t.Run("ZeroValues", func(t *testing.T) {
		t.Parallel()

		schema := Schema{}

		jsonData, err := json.Marshal(schema)
		require.NoError(t, err)

		var restored Schema
		err = json.Unmarshal(jsonData, &restored)
		require.NoError(t, err)

		assert.Equal(t, "", restored.UID)
		assert.Equal(t, "", restored.WriteKey)
		assert.Equal(t, "", restored.EventType)
		assert.Equal(t, "", restored.EventIdentifier)
		assert.Equal(t, 0, restored.Count)
		assert.True(t, restored.CreatedAt.IsZero())
		assert.True(t, restored.LastSeen.IsZero())
		assert.Nil(t, restored.Schema)
	})
}

func TestSchemasFile_JSONMarshaling(t *testing.T) {
	t.Parallel()

	t.Run("MarshalSchemasFile", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		schemasFile := SchemasFile{
			Schemas: []Schema{
				{
					UID:             "file-schema-1",
					WriteKey:        "file-key-1",
					EventType:       "track",
					EventIdentifier: "file_event_1",
					Schema: map[string]interface{}{
						"userId": "string",
						"event":  "string",
					},
					CreatedAt: now,
					LastSeen:  now,
					Count:     5,
				},
				{
					UID:             "file-schema-2",
					WriteKey:        "file-key-2",
					EventType:       "page",
					EventIdentifier: "file_event_2",
					Schema: map[string]interface{}{
						"anonymousId": "string",
						"properties": map[string]interface{}{
							"url":   "string",
							"title": "string",
						},
					},
					CreatedAt: now,
					LastSeen:  now,
					Count:     10,
				},
			},
		}

		jsonData, err := json.Marshal(schemasFile)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		jsonString := string(jsonData)
		assert.Contains(t, jsonString, "file-schema-1")
		assert.Contains(t, jsonString, "file-schema-2")
		assert.Contains(t, jsonString, "file-key-1")
		assert.Contains(t, jsonString, "file-key-2")
		assert.Contains(t, jsonString, "schemas")
	})

	t.Run("UnmarshalSchemasFile", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"schemas": [
				{
					"uid": "unmarshal-test-1",
					"writeKey": "unmarshal-key-1",
					"eventType": "track",
					"eventIdentifier": "unmarshal_event",
					"schema": {
						"userId": "string",
						"properties": {
							"test_prop": "string"
						}
					},
					"createdAt": "2024-01-01T12:00:00Z",
					"lastSeen": "2024-01-02T12:00:00Z",
					"count": 7
				}
			]
		}`

		var schemasFile SchemasFile
		err := json.Unmarshal([]byte(jsonData), &schemasFile)
		require.NoError(t, err)

		assert.Len(t, schemasFile.Schemas, 1)

		schema := schemasFile.Schemas[0]
		assert.Equal(t, "unmarshal-test-1", schema.UID)
		assert.Equal(t, "unmarshal-key-1", schema.WriteKey)
		assert.Equal(t, "track", schema.EventType)
		assert.Equal(t, "unmarshal_event", schema.EventIdentifier)
		assert.Equal(t, 7, schema.Count)

		expectedCreatedAt, err := time.Parse(time.RFC3339, "2024-01-01T12:00:00Z")
		require.NoError(t, err)
		assert.True(t, expectedCreatedAt.Equal(schema.CreatedAt))

		expectedLastSeen, err := time.Parse(time.RFC3339, "2024-01-02T12:00:00Z")
		require.NoError(t, err)
		assert.True(t, expectedLastSeen.Equal(schema.LastSeen))
	})

	t.Run("EmptyFile", func(t *testing.T) {
		t.Parallel()

		schemasFile := SchemasFile{
			Schemas: []Schema{},
		}

		jsonData, err := json.Marshal(schemasFile)
		require.NoError(t, err)

		var restored SchemasFile
		err = json.Unmarshal(jsonData, &restored)
		require.NoError(t, err)

		assert.Empty(t, restored.Schemas)
	})

	t.Run("NilSchemas", func(t *testing.T) {
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

func TestSchema_TimeHandling(t *testing.T) {
	t.Parallel()

	t.Run("DifferentTimeFormats", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name     string
			jsonTime string
		}{
			{
				name:     "RFC3339",
				jsonTime: "2024-01-10T10:08:15Z",
			},
			{
				name:     "RFC3339WithMilliseconds",
				jsonTime: "2024-01-10T10:08:15.407Z",
			},
			{
				name:     "RFC3339WithMicroseconds",
				jsonTime: "2024-01-10T10:08:15.407491Z",
			},
			{
				name:     "RFC3339WithTimezone",
				jsonTime: "2024-01-10T10:08:15+05:30",
			},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()

				jsonData := `{
					"uid": "time-format-test",
					"writeKey": "time-key",
					"eventType": "track",
					"eventIdentifier": "time_event",
					"schema": {"userId": "string"},
					"createdAt": "` + c.jsonTime + `",
					"lastSeen": "` + c.jsonTime + `",
					"count": 1
				}`

				var schema Schema
				err := json.Unmarshal([]byte(jsonData), &schema)
				require.NoError(t, err, "Should handle time format: %s", c.jsonTime)

				assert.Equal(t, "time-format-test", schema.UID)
				assert.False(t, schema.CreatedAt.IsZero())
				assert.False(t, schema.LastSeen.IsZero())
			})
		}
	})
}
