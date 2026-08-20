package bq

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	// schema.json uses negative lookaheads for both namespace and bucket names.
	// RE2 cannot compile lookaheads, so the disallowed cases live in reject patterns.
	funcs.NewPatternWithReject(
		"bq_namespace",
		`^(.{0,64})$`,
		`^(pg_|PG_|pG_|Pg_)`,
		"must be at most 64 characters, must not contain line breaks, and must not start with a pg_ prefix",
	)

	funcs.NewPatternWithReject(
		"bq_bucket_name",
		`^[a-z0-9][a-z0-9-._]{1,61}[a-z0-9]$`,
		`(^goog)|google|^\d+\.\d+\.\d+\.\d+$|\.\.`,
		"must be a valid GCS bucket name, must not start with goog, contain google, look like an IP address, or contain consecutive dots",
	)
}

// Source types from integrations-config destinations/bq/db-config.json.
var sourceTypes = []string{
	common.SourceTypeAndroid,
	common.SourceTypeAndroidKotlin,
	common.SourceTypeIOS,
	common.SourceTypeIOSSwift,
	common.SourceTypeWeb,
	common.SourceTypeUnity,
	common.SourceTypeAMP,
	common.SourceTypeCloud,
	common.SourceTypeReactNative,
	common.SourceTypeCloudSource,
	common.SourceTypeFlutter,
	common.SourceTypeCordova,
	common.SourceTypeShopify,
}

var connectionModes = map[string][]string{
	common.SourceTypeAndroid:       {"cloud"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeAMP:           {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeCloudSource:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeShopify:       {"cloud"},
}

// excludeWindow mirrors the only genuinely nested object in the upstream config.
type excludeWindow struct {
	StartTime string `mapstructure:"start_time" validate:"required"`
	EndTime   string `mapstructure:"end_time" validate:"required"`
}

// bqConfig is the local YAML config model. It is flat because the upstream BigQuery
// warehouse config is flat except excludeWindow; terraform's sync list is a
// provider-specific artifact.
type bqConfig struct {
	Project     string `mapstructure:"project" validate:"required,dynamic_or_pattern=single_line_100"`
	Location    string `mapstructure:"location" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	BucketName  string `mapstructure:"bucket_name" validate:"required,dynamic_or_pattern=bq_bucket_name"`
	Prefix      string `mapstructure:"prefix" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Namespace   string `mapstructure:"namespace" validate:"omitempty,dynamic_or_pattern=bq_namespace"`
	Credentials string `mapstructure:"credentials" validate:"required"`

	SyncFrequency string         `mapstructure:"sync_frequency" validate:"required,dynamic_or_oneof=5 10 15 30 60 180 360 720 1440"`
	SyncStartAt   string         `mapstructure:"sync_start_at"`
	ExcludeWindow *excludeWindow `mapstructure:"exclude_window"`

	SkipTracksTable           *bool                    `mapstructure:"skip_tracks_table"`
	SkipViews                 *bool                    `mapstructure:"skip_views"`
	SkipUsersTable            *bool                    `mapstructure:"skip_users_table"`
	PartitionColumn           string                   `mapstructure:"partition_column" validate:"omitempty,dynamic_or_oneof=_PARTITIONTIME loaded_at received_at timestamp sent_at original_timestamp"`
	PartitionType             string                   `mapstructure:"partition_type" validate:"omitempty,dynamic_or_oneof=hour day"`
	JSONPaths                 string                   `mapstructure:"json_paths"`
	CleanupObjectStorageFiles *bool                    `mapstructure:"cleanup_object_storage_files"`
	UnderscoreDivideNumbers   *bool                    `mapstructure:"underscore_divide_numbers"`
	AllowUsersContextTraits   *bool                    `mapstructure:"allow_users_context_traits"`
	ConsentManagement         common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the BigQuery warehouse destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("project", "project"),
		converter.Simple("location", "location"),
		converter.Simple("bucketName", "bucket_name"),
		converter.Simple("prefix", "prefix"),
		converter.Simple("namespace", "namespace"),
		converter.Simple("credentials", "credentials"),
		converter.Simple("syncFrequency", "sync_frequency"),
		converter.Simple("syncStartAt", "sync_start_at"),
		converter.Simple("excludeWindow.excludeWindowStartTime", "exclude_window.start_time"),
		converter.Simple("excludeWindow.excludeWindowEndTime", "exclude_window.end_time"),
		converter.Simple("skipTracksTable", "skip_tracks_table"),
		converter.Simple("skipViews", "skip_views"),
		converter.Simple("skipUsersTable", "skip_users_table"),
		converter.Simple("partitionColumn", "partition_column"),
		converter.Simple("partitionType", "partition_type"),
		converter.Simple("jsonPaths", "json_paths"),
		converter.Simple("cleanupObjectStorageFiles", "cleanup_object_storage_files"),
		converter.Simple("underscoreDivideNumbers", "underscore_divide_numbers"),
		converter.Simple("allowUsersContextTraits", "allow_users_context_traits"),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "bq",
		APIType:    "BQ",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"credentials"},
		NewConfig: func() any {
			return &bqConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
