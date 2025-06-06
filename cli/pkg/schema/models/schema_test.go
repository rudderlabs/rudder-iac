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

		response := SchemasResponse{
			Results: []Schema{
				{
					UID:             "test-uid-123",
					WriteKey:        "test-write-key",
					EventType:       "track",
					EventIdentifier: "product_viewed",
					Schema: map[string]interface{}{
						"userId": "string",
						"event":  "string",
					},
					CreatedAt: time.Date(2024, 1, 10, 10, 8, 15, 407491000, time.UTC),
					LastSeen:  time.Date(2024, 3, 25, 18, 49, 31, 870834000, time.UTC),
					Count:     42,
				},
			},
			CurrentPage: 1,
			HasNext:     true,
		}

		jsonData, err := json.Marshal(response)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		jsonString := string(jsonData)
		assert.Contains(t, jsonString, "test-uid-123")
		assert.Contains(t, jsonString, "currentPage")
		assert.Contains(t, jsonString, "hasNext")
		assert.Contains(t, jsonString, "results")
	})

	t.Run("UnmarshalFromJSON", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"results": [
				{
					"uid": "unmarshal-test",
					"writeKey": "unmarshal-key",
					"eventType": "track",
					"eventIdentifier": "test_event",
					"schema": {"userId": "string"},
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
		assert.Equal(t, "unmarshal-test", schema.UID)
		assert.Equal(t, "unmarshal-key", schema.WriteKey)
		assert.Equal(t, "test_event", schema.EventIdentifier)
		assert.Equal(t, 5, schema.Count)
	})
}

func TestSchema_JSONMarshaling(t *testing.T) {
	t.Parallel()

	t.Run("MarshalWithTimeFields", func(t *testing.T) {
		t.Parallel()

		schema := Schema{
			UID:             "test-uid",
			WriteKey:        "test-key",
			EventType:       "track",
			EventIdentifier: "purchase",
			Schema: map[string]interface{}{
				"userId": "string",
				"properties": map[string]interface{}{
					"amount": "number",
				},
			},
			CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			LastSeen:  time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
			Count:     10,
		}

		jsonData, err := json.Marshal(schema)
		require.NoError(t, err)

		var restored Schema
		err = json.Unmarshal(jsonData, &restored)
		require.NoError(t, err)

		assert.Equal(t, schema.UID, restored.UID)
		assert.Equal(t, schema.WriteKey, restored.WriteKey)
		assert.Equal(t, schema.EventIdentifier, restored.EventIdentifier)
		assert.True(t, schema.CreatedAt.Equal(restored.CreatedAt))
		assert.True(t, schema.LastSeen.Equal(restored.LastSeen))
	})
}

func TestSchemasFile_JSONMarshaling(t *testing.T) {
	t.Parallel()

	t.Run("MarshalSchemasFile", func(t *testing.T) {
		t.Parallel()

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
					Count: 5,
				},
			},
		}

		jsonData, err := json.Marshal(schemasFile)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		jsonString := string(jsonData)
		assert.Contains(t, jsonString, "file-schema-1")
		assert.Contains(t, jsonString, "file-key-1")
		assert.Contains(t, jsonString, "schemas")
	})
}
