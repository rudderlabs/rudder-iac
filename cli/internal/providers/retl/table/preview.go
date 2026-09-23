package table

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
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
	if t.AccountID == "" {
		return nil, fmt.Errorf("account ID not found in resource data")
	}
	return sqlmodel.PreviewQuery(ctx, h.client, &retlClient.PreviewSubmitRequest{
		SQL:       sql,
		AccountID: t.AccountID,
		Limit:     limit,
	})
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
		// Not ErrPreviewUnsupported: the definition IS supported as a source,
		// only its identifier quoting is unknown here. Reporting it as
		// unsupported would send the reader looking for a missing feature
		// rather than for this switch. Reachable only by adding a warehouse to
		// TableSpec's oneof tag without adding it here; collapsing the three
		// enumerations into one is DEX-877.
		return "", fmt.Errorf("preview cannot quote identifiers for source_definition %q: add its quoting to previewSQL", t.SourceDefinition)
	}
	return "select * from " + relation + " limit " + strconv.Itoa(max(limit, 1)), nil
}

func quote(name, q string) string {
	return q + strings.ReplaceAll(name, q, q+q) + q
}
