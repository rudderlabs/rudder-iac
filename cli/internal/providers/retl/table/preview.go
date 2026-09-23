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

// errNoQueryToPreview is the refusal for a table source that has no warehouse
// query. It was a sentinel while `validate` previewed and had to tell "nothing
// to run" apart from "the query failed"; validate no longer previews, so no
// caller matches on it and a plain error carries the same text.
var errNoQueryToPreview = errors.New("preview is not supported")

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
// A limit of 0 still reads one row, so a caller asking only "can this be read?"
// gets an answer without pulling data back.
func previewSQL(t TableSpec, limit int) (string, error) {
	if t.isS3() {
		return "", fmt.Errorf("%w for s3 table sources", errNoQueryToPreview)
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
		// Reachable only if TableSpec's oneof gains a warehouse this switch lacks;
		// collapsing the three enumerations into one is DEX-877.
		return "", fmt.Errorf("preview cannot quote identifiers for source_definition %q: add its quoting to previewSQL", t.SourceDefinition)
	}
	return "select * from " + relation + " limit " + strconv.Itoa(max(limit, 1)), nil
}

func quote(name, q string) string {
	return q + strings.ReplaceAll(name, q, q+q) + q
}
