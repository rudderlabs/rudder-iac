package validate_test

import (
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/pkg/validate"
	"github.com/rudderlabs/rudder-iac/cli/pkg/validate/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogValidator_Success(t *testing.T) {
	t.Parallel()

	catalog, err := localcatalog.Read("./testdata/valid-catalog")
	require.Nil(t, err)

	cv := validate.NewCatalogValidator()
	errs := cv.Validate(catalog)

	require.Len(t, errs, 0)
	require.Equal(t, errs.Error(), "")
}

func TestCatalogValidator_Failure(t *testing.T) {
	t.Parallel()

	catalog, err := localcatalog.Read("./testdata/invalid-catalog")
	require.Nil(t, err)

	cv := validate.NewCatalogValidator()
	errs := cv.Validate(catalog)

	require.Len(t, errs, 10)
	assert.Equal(t, errs, validate.ValidationErrors{
		{
			EntityType: entity.Event,
			Reference:  "#/events/app_events/user_signed_up",
			Err:        entity.ErrDuplicateByName,
		},
		{
			EntityType: entity.Event,
			Reference:  "#/events/app_events/user_signed_out",
			Err:        entity.ErrDuplicateByName,
		},
		{
			EntityType: entity.Event,
			Reference:  "#/events/app_events/page",
			Err:        entity.ErrNotAllowedKeyName,
		},
		{
			EntityType: entity.Event,
			Reference:  "#/events/app_events/alias",
			Err:        entity.ErrInvalidRequiredKeysEventType,
		},
		{
			EntityType: entity.Event,
			Reference:  "#/events/app_events/",
			Err:        entity.ErrMissingRequiredKeysID,
		},
		{
			EntityType: entity.Property,
			Reference:  "#/properties/app_properties/",
			Err:        entity.ErrMissingRequiredKeysID,
		},
		{
			EntityType: entity.Property,
			Reference:  "#/properties/app_properties/user_id",
			Err:        entity.ErrMissingRequiredKeysName,
		},
		{
			EntityType: entity.Property,
			Reference:  "#/properties/app_properties/user_id",
			Err:        entity.ErrInvalidRequiredKeysPropertyType,
		},
		{
			EntityType: entity.TrackingPlan,
			Reference:  "#/tp/trackingplan/first_tracking_plan",
			Err:        fmt.Errorf("%w: rule: %s property: %s", entity.ErrMissingEntityFromRef, "rule_01", "#/properties/app_properties/write_key"),
		},
		{
			EntityType: entity.TrackingPlan,
			Reference:  "#/tp/trackingplan/first_tracking_plan",
			Err:        fmt.Errorf("%w: rule: %s event: %s", entity.ErrMissingIdentityApplied, "rule_02", "#/events/app_events/page"),
		},
	},
	)
}
