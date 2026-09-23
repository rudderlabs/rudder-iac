package table_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
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
		{"postgres", "postgres", "public", "users", `select * from "public"."users" limit 10`},
		{"redshift", "redshift", "public", "users", `select * from "public"."users" limit 10`},
		{"snowflake", "snowflake", "public", "users", `select * from "public"."users" limit 10`},
		{"trino", "trino", "public", "users", `select * from "public"."users" limit 10`},
		{"mysql", "mysql", "public", "users", "select * from `public`.`users` limit 10"},
		{"databricks", "databricks", "public", "users", "select * from `public`.`users` limit 10"},
		{"bigquery", "bigquery", "public", "users", "select * from `public.users` limit 10"},

		{"double quotes keep case and are doubled", "snowflake", `My"Schema`, "Users Table", `select * from "My""Schema"."Users Table" limit 10`},
		{"backticks are doubled", "databricks", "raw", "we`ird", "select * from `raw`.`we``ird` limit 10"},
		{"bigquery escapes with backslashes", "bigquery", "raw", "we`ird'\"", "select * from `raw.we\\`ird\\'\\\"` limit 10"},
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

// validate previews with limit 0, which the request omits just as it does for
// SQL models; the query itself still stays bounded.
func TestPreviewWithoutLimitReadsOneRow(t *testing.T) {
	t.Parallel()
	store := &previewStore{}
	h, r := loadResource(t, store, warehouseSpec())

	_, err := h.Preview(context.Background(), r.ID(), r.Data(), 0)

	require.NoError(t, err)
	assert.Equal(t, []retlClient.PreviewSubmitRequest{{AccountID: "acc-123", SQL: `select * from "public"."users" limit 1`}}, store.requests)
}

func TestPreviewRejectsWithoutCallingTheAPI(t *testing.T) {
	t.Parallel()

	t.Run("s3 source", func(t *testing.T) {
		t.Parallel()
		store := &previewStore{}
		h, r := loadResource(t, store, s3Spec())

		_, err := h.Preview(context.Background(), r.ID(), r.Data(), 10)

		require.EqualError(t, err, "preview is not supported for s3 table sources")
		assert.ErrorIs(t, err, table.ErrPreviewUnsupported)
		assert.Empty(t, store.requests)
	})

	// #851 puts a PropertyRef under the account key when the spec references an
	// account, and preview runs before the ref resolves to a remote id.

	// The same ref stored by value, as datacatalog stores it. Matching only the
	// pointer form would fall through to "account ID not found in resource data".

	t.Run("unknown source definition", func(t *testing.T) {
		t.Parallel()
		store := &previewStore{}
		h := table.NewHandler(store, "retl")

		_, err := h.Preview(context.Background(), "users-table", resources.ResourceData{
			sqlmodel.AccountIDKey:        "acc-123",
			sqlmodel.SourceDefinitionKey: "oracle",
			table.SchemaKey:              "public",
			table.TableKey:               "users",
		}, 10)

		require.EqualError(t, err, `preview cannot quote identifiers for source_definition "oracle": add its quoting to previewSQL`)
		// Deliberately NOT ErrPreviewUnsupported. That sentinel means "this source
		// has no query to run", which `validate` now treats as a pass; an unknown
		// definition is a gap in previewSQL and must keep failing. Wrapping both in
		// one sentinel would let a missing quoting rule exit 0.
		assert.NotErrorIs(t, err, table.ErrPreviewUnsupported)
		assert.Empty(t, store.requests)
	})

	t.Run("bigquery name with a backslash", func(t *testing.T) {
		t.Parallel()
		spec := warehouseSpec()
		spec.Spec["source_definition"] = "bigquery"
		spec.Spec["table"] = "users\\` union select 1 --"
		store := &previewStore{}
		h, r := loadResource(t, store, spec)

		_, err := h.Preview(context.Background(), r.ID(), r.Data(), 10)

		require.EqualError(t, err, "bigquery schema and table names cannot contain a backslash: \"public.users\\\\` union select 1 --\"")
		assert.NotErrorIs(t, err, table.ErrPreviewUnsupported)
		assert.Empty(t, store.requests)
	})

	t.Run("missing account id", func(t *testing.T) {
		t.Parallel()
		store := &previewStore{}
		h := table.NewHandler(store, "retl")

		_, err := h.Preview(context.Background(), "users-table", resources.ResourceData{
			sqlmodel.SourceDefinitionKey: "postgres",
			table.SchemaKey:              "public",
			table.TableKey:               "users",
		}, 10)

		require.EqualError(t, err, "account ID not found in resource data")
		assert.Empty(t, store.requests)
	})
}
