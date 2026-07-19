package common_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func TestDestinationHTTPURLPattern(t *testing.T) {
	t.Parallel()

	type urlConfig struct {
		URL string `validate:"omitempty,pattern=destination_http_url"`
	}

	cases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "empty omitted", url: "", wantErr: false},
		{name: "https url", url: "https://www.googletagmanager.com", wantErr: false},
		{name: "http url", url: "http://gtm.example.com/path", wantErr: false},
		{name: "host without scheme", url: "www.googletagmanager.com", wantErr: false},
		{name: "env reference", url: "env.SDK_BASE_URL", wantErr: false},
		{name: "ui template", url: "{{ config.url || https://example.com }}", wantErr: false},
		{name: "iac variable", url: "{{ .SDK_BASE_URL }}", wantErr: false},
		{name: "ngrok https rejected", url: "https://abc123.ngrok.io", wantErr: true},
		{name: "ngrok path rejected", url: "https://abc123.ngrok.io/gtm", wantErr: true},
		{name: "invalid literal", url: "not a url", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			errs, err := rules.ValidateStruct(urlConfig{URL: tc.url}, "", funcs.GetPatternValidator())
			require.NoError(t, err)
			if tc.wantErr {
				require.NotEmpty(t, errs)
				return
			}
			assert.Empty(t, errs)
		})
	}
}

func TestDestinationHTTPURLPatternName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "destination_http_url", common.PatternDestinationHTTPURL)
}
