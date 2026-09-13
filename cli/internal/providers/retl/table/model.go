package table

import (
	"fmt"
	"slices"
	"strings"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

const (
	ResourceType = "retl-source-table"
	ResourceKind = "retl-source-table"
	ImportPath   = "tables"

	// Keys shared with sqlmodel alias its constants instead of redeclaring them:
	// a connection references either kind of RETL source through the same graph
	// data and output id, so the two source kinds must not drift apart.
	LocalIDKey          = sqlmodel.LocalIDKey
	DisplayNameKey      = sqlmodel.DisplayNameKey
	AccountIDKey        = sqlmodel.AccountIDKey
	PrimaryKeyKey       = sqlmodel.PrimaryKeyKey
	SourceDefinitionKey = sqlmodel.SourceDefinitionKey
	EnabledKey          = sqlmodel.EnabledKey
	IDKey               = sqlmodel.IDKey
	SourceTypeKey       = sqlmodel.SourceTypeKey
	CreatedAtKey        = sqlmodel.CreatedAtKey
	UpdatedAtKey        = sqlmodel.UpdatedAtKey

	SchemaKey       = "schema"
	TableKey        = "table"
	BucketNameKey   = "bucket_name"
	ObjectPrefixKey = "object_prefix"

	// SourceDefinitionS3 selects the bucket-backed config shape. The API also
	// exempts s3 sources from its primary key requirement.
	SourceDefinitionS3 = "s3"
)

// warehouseSourceDefinitions are the source definitions that take the
// schema/table config shape. They match the ones retl-source-sql-model accepts.
var warehouseSourceDefinitions = []string{
	string(sqlmodel.SourceDefinitionPostgres),
	string(sqlmodel.SourceDefinitionRedshift),
	string(sqlmodel.SourceDefinitionSnowflake),
	string(sqlmodel.SourceDefinitionBigQuery),
	string(sqlmodel.SourceDefinitionMySQL),
	string(sqlmodel.SourceDefinitionDatabricks),
	string(sqlmodel.SourceDefinitionTrino),
}

// TableSpec is the YAML shape of a retl-source-table spec. Warehouse and s3
// sources share one flat shape, as retl-source-sql-model does; validate decides
// which fields each source definition requires or forbids.
type TableSpec struct {
	ID               string `mapstructure:"id"`
	DisplayName      string `mapstructure:"display_name"`
	AccountID        string `mapstructure:"account_id"`
	SourceDefinition string `mapstructure:"source_definition"`
	PrimaryKey       string `mapstructure:"primary_key"`
	Schema           string `mapstructure:"schema"`
	Table            string `mapstructure:"table"`
	BucketName       string `mapstructure:"bucket_name"`
	ObjectPrefix     string `mapstructure:"object_prefix"`
	Enabled          *bool  `mapstructure:"enabled"`
}

func (s *TableSpec) validate() error {
	if missing := blankFields(map[string]string{
		IDKey:               s.ID,
		DisplayNameKey:      s.DisplayName,
		AccountIDKey:        s.AccountID,
		SourceDefinitionKey: s.SourceDefinition,
	}); len(missing) > 0 {
		return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
	}

	if s.SourceDefinition == SourceDefinitionS3 {
		return s.validateS3()
	}
	if !slices.Contains(warehouseSourceDefinitions, s.SourceDefinition) {
		return fmt.Errorf("invalid source_definition %q: must be one of %s, %s",
			s.SourceDefinition, strings.Join(warehouseSourceDefinitions, ", "), SourceDefinitionS3)
	}
	return s.validateWarehouse()
}

func (s *TableSpec) validateWarehouse() error {
	if missing := blankFields(map[string]string{
		PrimaryKeyKey: s.PrimaryKey,
		SchemaKey:     s.Schema,
		TableKey:      s.Table,
	}); len(missing) > 0 {
		return fmt.Errorf("missing required fields for source_definition %q: %s", s.SourceDefinition, strings.Join(missing, ", "))
	}
	if set := presentFields(map[string]string{
		BucketNameKey:   s.BucketName,
		ObjectPrefixKey: s.ObjectPrefix,
	}); len(set) > 0 {
		return fmt.Errorf("fields not supported for source_definition %q: %s", s.SourceDefinition, strings.Join(set, ", "))
	}
	return nil
}

// validateS3 rejects primary_key rather than ignoring it: the s3 config the API
// client sends has no primary key field, so an accepted value would never reach
// the server and would show as a change on every plan.
func (s *TableSpec) validateS3() error {
	if s.BucketName == "" {
		return fmt.Errorf("missing required fields for source_definition %q: %s", SourceDefinitionS3, BucketNameKey)
	}
	if set := presentFields(map[string]string{
		PrimaryKeyKey: s.PrimaryKey,
		SchemaKey:     s.Schema,
		TableKey:      s.Table,
	}); len(set) > 0 {
		return fmt.Errorf("fields not supported for source_definition %q: %s", SourceDefinitionS3, strings.Join(set, ", "))
	}
	return nil
}

// blankFields returns the sorted names of the empty values.
func blankFields(fields map[string]string) []string {
	var names []string
	for name, value := range fields {
		if value == "" {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// presentFields returns the sorted names of the non-empty values.
func presentFields(fields map[string]string) []string {
	var names []string
	for name, value := range fields {
		if value != "" {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// TableResource is a table source in the one shape that local specs and remote
// sources both reduce to, so plan input and remote state are built by the same
// code and compare field for field.
//
// There is no description: neither table config shape carries one, so it could
// never round-trip.
type TableResource struct {
	ID               string
	DisplayName      string
	AccountID        string
	SourceDefinition string
	PrimaryKey       string
	Schema           string
	Table            string
	BucketName       string
	ObjectPrefix     string
	Enabled          bool
}

func (t TableResource) isS3() bool {
	return t.SourceDefinition == SourceDefinitionS3
}

// configData returns the fields that belong to the source definition's config
// shape.
func (t TableResource) configData() resources.ResourceData {
	if t.isS3() {
		return resources.ResourceData{
			BucketNameKey:   t.BucketName,
			ObjectPrefixKey: t.ObjectPrefix,
		}
	}
	return resources.ResourceData{
		PrimaryKeyKey: t.PrimaryKey,
		SchemaKey:     t.Schema,
		TableKey:      t.Table,
	}
}

// data returns the resource's graph data, without the local id. primary_key is
// present for every source definition — empty for s3 — so consumers read one
// shape across both RETL source kinds.
func (t TableResource) data() resources.ResourceData {
	data := t.configData()
	data[DisplayNameKey] = t.DisplayName
	data[AccountIDKey] = t.AccountID
	data[SourceDefinitionKey] = t.SourceDefinition
	data[PrimaryKeyKey] = t.PrimaryKey
	data[EnabledKey] = t.Enabled
	return data
}

// specFields returns the flat spec body export writes for the resource.
func (t TableResource) specFields(id string) map[string]any {
	fields := t.configData()
	if t.isS3() && t.ObjectPrefix == "" {
		delete(fields, ObjectPrefixKey)
	}
	fields[IDKey] = id
	fields[DisplayNameKey] = t.DisplayName
	fields[AccountIDKey] = t.AccountID
	fields[SourceDefinitionKey] = t.SourceDefinition
	fields[EnabledKey] = t.Enabled
	return fields
}

func (t TableResource) config() retlClient.RETLConfig {
	if t.isS3() {
		return retlClient.RETLS3TableConfig{
			BucketName:   t.BucketName,
			ObjectPrefix: t.ObjectPrefix,
		}
	}
	return retlClient.RETLTableConfig{
		PrimaryKey: t.PrimaryKey,
		Schema:     t.Schema,
		Table:      t.Table,
	}
}

// fromData reads a resource back from graph or state data. Missing keys read as
// zero values: s3 data carries no schema/table and warehouse data no bucket.
func fromData(data resources.ResourceData) TableResource {
	str := func(key string) string {
		v, _ := data[key].(string)
		return v
	}
	enabled, _ := data[EnabledKey].(bool)
	return TableResource{
		ID:               str(LocalIDKey),
		DisplayName:      str(DisplayNameKey),
		AccountID:        str(AccountIDKey),
		SourceDefinition: str(SourceDefinitionKey),
		PrimaryKey:       str(PrimaryKeyKey),
		Schema:           str(SchemaKey),
		Table:            str(TableKey),
		BucketName:       str(BucketNameKey),
		ObjectPrefix:     str(ObjectPrefixKey),
		Enabled:          enabled,
	}
}

// fromRemote reads a resource from an API source. The local id is left empty:
// callers know whether it comes from the external id or the namer.
func fromRemote(source *retlClient.RETLSource) (TableResource, error) {
	t := TableResource{
		DisplayName:      source.Name,
		AccountID:        source.AccountID,
		SourceDefinition: source.SourceDefinitionName,
		Enabled:          source.IsEnabled,
	}
	if t.isS3() {
		cfg, err := retlClient.DecodeConfig[retlClient.RETLS3TableConfig](source.Config)
		if err != nil {
			return TableResource{}, fmt.Errorf("decoding s3 table config for source %s: %w", source.ID, err)
		}
		t.BucketName = cfg.BucketName
		t.ObjectPrefix = cfg.ObjectPrefix
		return t, nil
	}
	cfg, err := retlClient.DecodeConfig[retlClient.RETLTableConfig](source.Config)
	if err != nil {
		return TableResource{}, fmt.Errorf("decoding table config for source %s: %w", source.ID, err)
	}
	t.PrimaryKey = cfg.PrimaryKey
	t.Schema = cfg.Schema
	t.Table = cfg.Table
	return t, nil
}

// toOutput builds the state output for an API source. It carries the remote id
// under IDKey, which is what a PropertyRef to this source resolves.
func toOutput(source *retlClient.RETLSource) (*resources.ResourceData, error) {
	t, err := fromRemote(source)
	if err != nil {
		return nil, err
	}
	output := t.data()
	output[IDKey] = source.ID
	output[SourceTypeKey] = source.SourceType
	if source.CreatedAt != nil {
		output[CreatedAtKey] = source.CreatedAt
	}
	if source.UpdatedAt != nil {
		output[UpdatedAtKey] = source.UpdatedAt
	}
	return &output, nil
}
