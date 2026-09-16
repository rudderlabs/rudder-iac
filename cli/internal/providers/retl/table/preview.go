package table

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// ErrPreviewUnsupported marks a table source that has no warehouse query to
// preview, so callers can tell it apart from a query that failed to run.
var ErrPreviewUnsupported = errors.New("preview is not supported")

// bigQueryEscaper escapes quote characters the way BigQuery string literals
// do, which is also how its quoted identifiers are escaped.
var bigQueryEscaper = strings.NewReplacer(`'`, `\'`, `"`, `\"`, "`", "\\`")

// Preview runs a query over the source's table through the same preview API as
// SQL models. The limit travels in the request, as it does for SQL models, and
// bounds the query too, so reading a table never depends on the server
// capping the rows.
func (h *Handler) Preview(ctx context.Context, id string, data resources.ResourceData, limit int) ([]map[string]any, error) {
	t := fromData(data)
	sql, err := previewSQL(t, limit)
	if err != nil {
		return nil, err
	}
	// Preview reads the project graph without remote state, so the remote id a
	// referenced account resolves to is not known here. fromData reads the key
	// through a checked assert, so a reference reads back as an empty id and
	// would otherwise surface as "account ID not found".
	if _, ok := data[sqlmodel.AccountIDKey].(*resources.PropertyRef); ok {
		return nil, fmt.Errorf("preview does not support table sources that reference their account yet: set account_id on %s to preview it", id)
	}
	if t.AccountID == "" {
		return nil, fmt.Errorf("account ID not found in resource data")
	}
	return sqlmodel.PreviewQuery(ctx, h.client, t.AccountID, sql, limit)
}

// previewSQL quotes the schema and table as sqlconnect-go does when
// rudder-sources queries a table source, so the preview reads the same relation:
// quoted names keep their case and may contain any character.
//
// validate previews with limit 0; the query still reads one row so that it
// proves the table can be read, not just that it resolves.
func previewSQL(t TableSpec, limit int) (string, error) {
	if t.isS3() {
		return "", fmt.Errorf("%w for s3 table sources", ErrPreviewUnsupported)
	}

	var relation string
	switch sqlmodel.SourceDefinition(t.SourceDefinition) {
	case sqlmodel.SourceDefinitionPostgres, sqlmodel.SourceDefinitionRedshift,
		sqlmodel.SourceDefinitionSnowflake, sqlmodel.SourceDefinitionTrino:
		relation = quote(t.Schema, `"`) + "." + quote(t.Table, `"`)
	case sqlmodel.SourceDefinitionMySQL, sqlmodel.SourceDefinitionDatabricks:
		relation = quote(t.Schema, "`") + "." + quote(t.Table, "`")
	case sqlmodel.SourceDefinitionBigQuery:
		// A backslash would escape the escaping itself and could close the
		// identifier early. BigQuery dataset and table names cannot contain
		// one, so rejecting it loses nothing.
		path := t.Schema + "." + t.Table
		if strings.Contains(path, `\`) {
			return "", fmt.Errorf("bigquery schema and table names cannot contain a backslash: %q", path)
		}
		// BigQuery accepts the whole dataset.table path as one quoted
		// identifier, and escapes with backslashes rather than by doubling.
		relation = "`" + bigQueryEscaper.Replace(path) + "`"
	default:
		return "", fmt.Errorf("%w for source_definition %q", ErrPreviewUnsupported, t.SourceDefinition)
	}
	return "select * from " + relation + " limit " + strconv.Itoa(max(limit, 1)), nil
}

// quote wraps name in q, doubling any q inside it.
func quote(name, q string) string {
	return q + strings.ReplaceAll(name, q, q+q) + q
}
