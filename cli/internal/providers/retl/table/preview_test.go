package table_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// previewStore is a preview API that records each submitted request and
// completes it on the first poll.
type previewStore struct {
	retlClient.RETLStore
	requests []retlClient.PreviewSubmitRequest
}

func (s *previewStore) SubmitSourcePreview(_ context.Context, req *retlClient.PreviewSubmitRequest) (*retlClient.PreviewSubmitResponse, error) {
	s.requests = append(s.requests, *req)
	return &retlClient.PreviewSubmitResponse{ID: "req-1"}, nil
}

func (s *previewStore) GetSourcePreviewResult(_ context.Context, _ string) (*retlClient.PreviewResultResponse, error) {
	return &retlClient.PreviewResultResponse{
		Status: retlClient.Completed,
		Rows:   []map[string]any{{"id": 1}},
	}, nil
}

func TestPreviewQueriesTheTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		sourceDefinition string
		schema           string
		table            string
		want             string
	}{
		{"postgres", "postgres", "public", "users", `select * from "public"."users"`},
		{"redshift", "redshift", "public", "users", `select * from "public"."users"`},
		{"snowflake", "snowflake", "public", "users", `select * from "public"."users"`},
		{"trino", "trino", "public", "users", `select * from "public"."users"`},
		{"mysql", "mysql", "public", "users", "select * from `public`.`users`"},
		{"databricks", "databricks", "public", "users", "select * from `public`.`users`"},
		{"bigquery", "bigquery", "public", "users", "select * from `public.users`"},

		{"double quotes keep case and are doubled", "snowflake", `My"Schema`, "Users Table", `select * from "My""Schema"."Users Table"`},
		{"backticks are doubled", "databricks", "raw", "we`ird", "select * from `raw`.`we``ird`"},
		{"bigquery escapes with backslashes", "bigquery", "raw", "we`ird'\"", "select * from `raw.we\\`ird\\'\\\"`"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			spec := warehouseSpec()
			spec.Spec["source_definition"] = tc.sourceDefinition
			spec.Spec["schema"] = tc.schema
			spec.Spec["table"] = tc.table
			store := &previewStore{}
			h, r := loadResource(t, store, spec)

			rows, err := h.Preview(context.Background(), r.ID(), r.Data(), 10)

			require.NoError(t, err)
			assert.Equal(t, []map[string]any{{"id": 1}}, rows)
			assert.Equal(t, []retlClient.PreviewSubmitRequest{{AccountID: "acc-123", Limit: 10, SQL: tc.want}}, store.requests)
		})
	}
}

func TestPreviewRejectsWithoutCallingTheAPI(t *testing.T) {
	t.Parallel()

	t.Run("s3 source", func(t *testing.T) {
		t.Parallel()
		store := &previewStore{}
		h, r := loadResource(t, store, s3Spec())

		_, err := h.Preview(context.Background(), r.ID(), r.Data(), 10)

		require.EqualError(t, err, "preview is not supported for s3 table sources")
		assert.Empty(t, store.requests)
	})

	t.Run("unknown source definition", func(t *testing.T) {
		t.Parallel()
		store := &previewStore{}
		h := table.NewHandler(store, "retl")

		_, err := h.Preview(context.Background(), "users-table", resources.ResourceData{
			table.AccountIDKey:        "acc-123",
			table.SourceDefinitionKey: "oracle",
			table.SchemaKey:           "public",
			table.TableKey:            "users",
		}, 10)

		require.EqualError(t, err, `preview is not supported for source_definition "oracle"`)
		assert.Empty(t, store.requests)
	})

	t.Run("missing account id", func(t *testing.T) {
		t.Parallel()
		store := &previewStore{}
		h := table.NewHandler(store, "retl")

		_, err := h.Preview(context.Background(), "users-table", resources.ResourceData{
			table.SourceDefinitionKey: "postgres",
			table.SchemaKey:           "public",
			table.TableKey:            "users",
		}, 10)

		require.EqualError(t, err, "account ID not found in resource data")
		assert.Empty(t, store.requests)
	})
}
