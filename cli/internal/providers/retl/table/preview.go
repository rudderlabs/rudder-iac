package table

import (
	"context"
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// bigQueryEscaper escapes quote characters the way BigQuery string literals
// do, which is also how its quoted identifiers are escaped.
var bigQueryEscaper = strings.NewReplacer(`'`, `\'`, `"`, `\"`, "`", "\\`")

// Preview runs a query over the source's whole table through the same preview
// API as SQL models. The row limit travels in the request, as it does for SQL
// models, rather than in the query.
func (h *Handler) Preview(ctx context.Context, _ string, data resources.ResourceData, limit int) ([]map[string]any, error) {
	t := fromData(data)
	sql, err := previewSQL(t)
	if err != nil {
		return nil, err
	}
	if t.AccountID == "" {
		return nil, fmt.Errorf("account ID not found in resource data")
	}
	return sqlmodel.PreviewQuery(ctx, h.client, t.AccountID, sql, limit)
}

// previewSQL quotes the schema and table as sqlconnect-go does when
// rudder-sources queries a table source, so the preview reads the same relation:
// quoted names keep their case and may contain any character.
func previewSQL(t TableResource) (string, error) {
	if t.isS3() {
		return "", fmt.Errorf("preview is not supported for s3 table sources")
	}

	var relation string
	switch sqlmodel.SourceDefinition(t.SourceDefinition) {
	case sqlmodel.SourceDefinitionPostgres, sqlmodel.SourceDefinitionRedshift,
		sqlmodel.SourceDefinitionSnowflake, sqlmodel.SourceDefinitionTrino:
		relation = quote(t.Schema, `"`) + "." + quote(t.Table, `"`)
	case sqlmodel.SourceDefinitionMySQL, sqlmodel.SourceDefinitionDatabricks:
		relation = quote(t.Schema, "`") + "." + quote(t.Table, "`")
	case sqlmodel.SourceDefinitionBigQuery:
		// BigQuery accepts the whole dataset.table path as one quoted
		// identifier, and escapes with backslashes rather than by doubling.
		relation = "`" + bigQueryEscaper.Replace(t.Schema+"."+t.Table) + "`"
	default:
		return "", fmt.Errorf("preview is not supported for source_definition %q", t.SourceDefinition)
	}
	return "select * from " + relation, nil
}

// quote wraps name in q, doubling any q inside it.
func quote(name, q string) string {
	return q + strings.ReplaceAll(name, q, q+q) + q
}
