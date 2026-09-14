package table

import (
	"fmt"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

const (
	ResourceType = "retl-source-table"
	ResourceKind = "retl-source-table"
	ImportPath   = "tables"

	SchemaKey       = "schema"
	TableKey        = "table"
	BucketNameKey   = "bucket_name"
	ObjectPrefixKey = "object_prefix"

	// SourceDefinitionS3 selects the bucket-backed config shape. The API also
	// exempts s3 sources from its primary key requirement.
	SourceDefinitionS3 = "s3"
)

// TableSpec is a table source in the one shape that local specs and remote
// sources both reduce to, so plan input and remote state are built by the same
// code and compare field for field. Warehouse and s3 sources share one flat
// shape, as retl-source-sql-model does; the validate tags, enforced by the
// retl/table/spec-syntax-valid rule, decide which fields each source definition
// requires or forbids.
//
// There is no description: neither table config shape carries one, so it could
// never round-trip. s3 forbids primary_key for the same reason: the s3 config
// has no field for it, so an accepted value would show as a change on every
// plan.
type TableSpec struct {
	ID               string `json:"id"                mapstructure:"id"                validate:"required"`
	DisplayName      string `json:"display_name"      mapstructure:"display_name"      validate:"required"`
	AccountID        string `json:"account_id"        mapstructure:"account_id"        validate:"required"`
	SourceDefinition string `json:"source_definition" mapstructure:"source_definition" validate:"required,oneof=postgres redshift snowflake bigquery mysql databricks trino s3"`
	PrimaryKey       string `json:"primary_key"       mapstructure:"primary_key"       validate:"required_unless=SourceDefinition s3,excluded_if=SourceDefinition s3"`
	Schema           string `json:"schema"            mapstructure:"schema"            validate:"required_unless=SourceDefinition s3,excluded_if=SourceDefinition s3"`
	Table            string `json:"table"             mapstructure:"table"             validate:"required_unless=SourceDefinition s3,excluded_if=SourceDefinition s3"`
	BucketName       string `json:"bucket_name"       mapstructure:"bucket_name"       validate:"required_if=SourceDefinition s3,excluded_unless=SourceDefinition s3"`
	ObjectPrefix     string `json:"object_prefix"     mapstructure:"object_prefix"     validate:"excluded_unless=SourceDefinition s3"`
	Enabled          bool   `json:"enabled"           mapstructure:"enabled"`
}

func (t TableSpec) isS3() bool {
	return t.SourceDefinition == SourceDefinitionS3
}

// configData returns the fields that belong to the source definition's config
// shape.
func (t TableSpec) configData() resources.ResourceData {
	if t.isS3() {
		return resources.ResourceData{
			BucketNameKey:   t.BucketName,
			ObjectPrefixKey: t.ObjectPrefix,
		}
	}
	return resources.ResourceData{
		sqlmodel.PrimaryKeyKey: t.PrimaryKey,
		SchemaKey:              t.Schema,
		TableKey:               t.Table,
	}
}

// data returns the resource's graph data, without the local id. The keys are
// sqlmodel's and primary_key is present for every source definition — empty
// for s3 — so a connection reads either RETL source kind through one shape.
func (t TableSpec) data() resources.ResourceData {
	data := t.configData()
	data[sqlmodel.DisplayNameKey] = t.DisplayName
	data[sqlmodel.AccountIDKey] = t.AccountID
	data[sqlmodel.SourceDefinitionKey] = t.SourceDefinition
	data[sqlmodel.PrimaryKeyKey] = t.PrimaryKey
	data[sqlmodel.EnabledKey] = t.Enabled
	return data
}

// specFields returns the flat spec body export writes for the resource.
func (t TableSpec) specFields(id string) map[string]any {
	fields := t.configData()
	if t.isS3() && t.ObjectPrefix == "" {
		delete(fields, ObjectPrefixKey)
	}
	fields[sqlmodel.IDKey] = id
	fields[sqlmodel.DisplayNameKey] = t.DisplayName
	fields[sqlmodel.AccountIDKey] = t.AccountID
	fields[sqlmodel.SourceDefinitionKey] = t.SourceDefinition
	fields[sqlmodel.EnabledKey] = t.Enabled
	return fields
}

func (t TableSpec) config() retlClient.RETLConfig {
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
func fromData(data resources.ResourceData) TableSpec {
	str := func(key string) string {
		v, _ := data[key].(string)
		return v
	}
	enabled, _ := data[sqlmodel.EnabledKey].(bool)
	return TableSpec{
		ID:               str(sqlmodel.LocalIDKey),
		DisplayName:      str(sqlmodel.DisplayNameKey),
		AccountID:        str(sqlmodel.AccountIDKey),
		SourceDefinition: str(sqlmodel.SourceDefinitionKey),
		PrimaryKey:       str(sqlmodel.PrimaryKeyKey),
		Schema:           str(SchemaKey),
		Table:            str(TableKey),
		BucketName:       str(BucketNameKey),
		ObjectPrefix:     str(ObjectPrefixKey),
		Enabled:          enabled,
	}
}

// fromRemote reads a resource from an API source. The local id is left empty:
// callers know whether it comes from the external id or the namer.
func fromRemote(source *retlClient.RETLSource) (TableSpec, error) {
	t := TableSpec{
		DisplayName:      source.Name,
		AccountID:        source.AccountID,
		SourceDefinition: source.SourceDefinitionName,
		Enabled:          source.IsEnabled,
	}
	if t.isS3() {
		cfg, err := retlClient.DecodeConfig[retlClient.RETLS3TableConfig](source.Config)
		if err != nil {
			return TableSpec{}, fmt.Errorf("decoding s3 table config for source %s: %w", source.ID, err)
		}
		t.BucketName = cfg.BucketName
		t.ObjectPrefix = cfg.ObjectPrefix
		return t, nil
	}
	cfg, err := retlClient.DecodeConfig[retlClient.RETLTableConfig](source.Config)
	if err != nil {
		return TableSpec{}, fmt.Errorf("decoding table config for source %s: %w", source.ID, err)
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
	output[sqlmodel.IDKey] = source.ID
	output[sqlmodel.SourceTypeKey] = source.SourceType
	if source.CreatedAt != nil {
		output[sqlmodel.CreatedAtKey] = source.CreatedAt
	}
	if source.UpdatedAt != nil {
		output[sqlmodel.UpdatedAtKey] = source.UpdatedAt
	}
	return &output, nil
}
