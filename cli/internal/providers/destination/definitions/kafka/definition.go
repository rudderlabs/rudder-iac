package kafka

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

const (
	kafkaHostNamePattern     = `^(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*(?:,\s*(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*)*$`
	kafkaPortPattern         = `^([1-9]|[1-9][0-9]{1,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])$`
	kafkaTopicPattern        = `^[a-zA-Z0-9_.\-]{1,249}$`
	kafkaSSHHostPattern      = `^(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-.]{0,251}[a-zA-Z0-9])(?::\d{1,5})?$`
	kafkaSSHUserPattern      = `^[a-zA-Z0-9_-]{1,32}$`
	kafkaSSHPublicKeyPattern = `^ssh-(rsa|ed25519|dss) [A-Za-z0-9+/]+[=]{0,3}( [^@]+@[^@]+)?$`
	kafkaMappingValuePattern = `^(.{0,100})$`
)

func init() {
	funcs.NewPattern("kafka_host_name", kafkaHostNamePattern, "must be one or more comma-separated host names")
	funcs.NewPattern("kafka_port", kafkaPortPattern, "must be a string integer from 1 to 65535")
	funcs.NewPattern("kafka_topic", kafkaTopicPattern, "must be 1-249 characters and contain only letters, digits, underscores, periods, and hyphens")
	funcs.NewPattern("kafka_ssh_host", kafkaSSHHostPattern, "must be a host name with an optional port")
	funcs.NewPattern("kafka_ssh_user", kafkaSSHUserPattern, "must be 1-32 characters and contain only letters, digits, underscores, and hyphens")
	funcs.NewPattern("kafka_ssh_public_key", kafkaSSHPublicKeyPattern, "must be an ssh-rsa, ssh-ed25519, or ssh-dss public key")
	funcs.NewPattern("kafka_mapping_value", kafkaMappingValuePattern, "must be at most 100 characters and must not contain line breaks")
}

// Source types from integrations-config destinations/kafka/db-config.json.
var sourceTypes = []string{
	common.SourceTypeAndroid,
	common.SourceTypeAndroidKotlin,
	common.SourceTypeIOS,
	common.SourceTypeIOSSwift,
	common.SourceTypeWeb,
	common.SourceTypeUnity,
	common.SourceTypeAMP,
	common.SourceTypeCloud,
	common.SourceTypeWarehouse,
	common.SourceTypeReactNative,
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
	common.SourceTypeWarehouse:     {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeShopify:       {"cloud"},
}

// kafkaConfig is the local YAML config model. Field set mirrors
// integrations-config destinations/kafka defaultConfig; validations mirror
// schema.json, including Kafka's SSL/SASL, Avro, multi-topic, and SSH settings.
type kafkaConfig struct {
	HostName            string                   `mapstructure:"host_name" validate:"required,dynamic_or_pattern=kafka_host_name"`
	Port                string                   `mapstructure:"port" validate:"required,dynamic_or_pattern=kafka_port"`
	Topic               string                   `mapstructure:"topic" validate:"required,dynamic_or_pattern=kafka_topic"`
	SSLEnabled          *bool                    `mapstructure:"ssl_enabled"`
	CACertificate       string                   `mapstructure:"ca_certificate"`
	UseSASL             *bool                    `mapstructure:"use_sasl"`
	SASLType            string                   `mapstructure:"sasl_type" validate:"required_if=SSLEnabled true UseSASL true,omitempty,dynamic_or_oneof=plain sha256 sha512"`
	Username            string                   `mapstructure:"username" validate:"required_if=SSLEnabled true UseSASL true,omitempty,dynamic_or_pattern=kafka_ssh_user"`
	Password            string                   `mapstructure:"password" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	ConvertToAvro       *bool                    `mapstructure:"convert_to_avro"`
	AvroSchemas         []avroSchema             `mapstructure:"avro_schemas" validate:"required_if=ConvertToAvro true,omitempty,dive"`
	EnableMultiTopic    *bool                    `mapstructure:"enable_multi_topic"`
	EventTypeToTopicMap []eventTypeTopicMapping  `mapstructure:"event_type_to_topic_map" validate:"omitempty,dive"`
	EventToTopicMap     []topicMapping           `mapstructure:"event_to_topic_map" validate:"omitempty,dive"`
	UseSSH              *bool                    `mapstructure:"use_ssh"`
	SSHHost             string                   `mapstructure:"ssh_host" validate:"required_if=UseSSH true,omitempty,dynamic_or_pattern=kafka_ssh_host"`
	SSHPort             string                   `mapstructure:"ssh_port" validate:"required_if=UseSSH true,omitempty,dynamic_or_pattern=kafka_port"`
	SSHUser             string                   `mapstructure:"ssh_user" validate:"required_if=UseSSH true,omitempty,dynamic_or_pattern=kafka_ssh_user"`
	SSHPublicKey        string                   `mapstructure:"ssh_public_key" validate:"required_if=UseSSH true,omitempty,dynamic_or_pattern=kafka_ssh_public_key"`
	EmbedAvroSchemaID   *bool                    `mapstructure:"embed_avro_schema_id"`
	ConsentManagement   common.ConsentManagement `mapstructure:"consent_management"`
}

type avroSchema struct {
	SchemaID string `mapstructure:"schema_id" validate:"required"`
	Schema   string `mapstructure:"schema" validate:"required"`
}

type eventTypeTopicMapping struct {
	From string `mapstructure:"from" validate:"omitempty,dynamic_or_oneof=identify page screen group alias"`
	To   string `mapstructure:"to" validate:"omitempty,dynamic_or_pattern=kafka_mapping_value"`
}

type topicMapping struct {
	From string `mapstructure:"from" validate:"omitempty,dynamic_or_pattern=kafka_mapping_value"`
	To   string `mapstructure:"to" validate:"omitempty,dynamic_or_pattern=kafka_mapping_value"`
}

// NewDefinition returns the Apache Kafka destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("hostName", "host_name"),
		converter.Simple("port", "port"),
		converter.Simple("topic", "topic"),
		converter.Simple("sslEnabled", "ssl_enabled"),
		converter.Simple("caCertificate", "ca_certificate"),
		converter.Simple("useSASL", "use_sasl"),
		converter.Simple("saslType", "sasl_type"),
		converter.Simple("username", "username"),
		converter.Simple("password", "password"),
		converter.Simple("convertToAvro", "convert_to_avro"),
		converter.ArrayWithObjects("avroSchemas", "avro_schemas", map[string]any{
			"schemaId": "schema_id",
			"schema":   "schema",
		}),
		converter.Simple("embedAvroSchemaID", "embed_avro_schema_id"),
		converter.Simple("enableMultiTopic", "enable_multi_topic"),
		converter.ArrayWithObjects("eventTypeToTopicMap", "event_type_to_topic_map", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("eventToTopicMap", "event_to_topic_map", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.Simple("useSSH", "use_ssh"),
		converter.Simple("sshHost", "ssh_host"),
		converter.Simple("sshPort", "ssh_port"),
		converter.Simple("sshUser", "ssh_user"),
		converter.Simple("sshPublicKey", "ssh_public_key"),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "kafka",
		APIType:    "KAFKA",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"password"},
		NewConfig: func() any {
			return &kafkaConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
