package varsubst

import "testing"

func TestEnvResolver(t *testing.T) {
	t.Setenv("RUDDER_DB_HOST", "db.example.com")
	t.Setenv("RUDDER_db_host", "db-lower.example.com")
	t.Setenv("RUDDER_EMPTY", "")
	t.Setenv("OTHER_TOKEN", "ignore-me")
	t.Setenv("CFG_PORT", "5432")
	t.Setenv("CFG_HOST", "localhost")
	t.Setenv("NO_PREFIX_KEY", "no-prefix")

	testCases := []struct {
		name          string
		prefix        string
		variableName  string
		expectedValue string
		expectedFound bool
	}{
		{
			name:          "prefix stripping with default prefix",
			prefix:        "",
			variableName:  "DB_HOST",
			expectedValue: "db.example.com",
			expectedFound: true,
		},
		{
			name:          "case sensitivity preserved",
			prefix:        "",
			variableName:  "db_host",
			expectedValue: "db-lower.example.com",
			expectedFound: true,
		},
		{
			name:          "empty value is found",
			prefix:        "",
			variableName:  "EMPTY",
			expectedValue: "",
			expectedFound: true,
		},
		{
			name:          "missing variable returns not found",
			prefix:        "",
			variableName:  "DOES_NOT_EXIST",
			expectedValue: "",
			expectedFound: false,
		},
		{
			name:          "non matching prefix is ignored",
			prefix:        "",
			variableName:  "TOKEN",
			expectedValue: "",
			expectedFound: false,
		},
		{
			name:          "multiple env vars load with custom prefix",
			prefix:        "CFG_",
			variableName:  "PORT",
			expectedValue: "5432",
			expectedFound: true,
		},
		{
			name:          "multiple env vars load with custom prefix second key",
			prefix:        "CFG_",
			variableName:  "HOST",
			expectedValue: "localhost",
			expectedFound: true,
		},
		{
			name:          "default prefix does not load keys without matching prefix",
			prefix:        "",
			variableName:  "KEY",
			expectedValue: "",
			expectedFound: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			varResolver := NewEnvResolver(testCase.prefix)

			actualValue, actualFound := varResolver.Resolve(testCase.variableName)

			if actualFound != testCase.expectedFound {
				t.Fatalf("expected found=%t, got found=%t", testCase.expectedFound, actualFound)
			}
			if actualValue != testCase.expectedValue {
				t.Fatalf("expected value=%q, got value=%q", testCase.expectedValue, actualValue)
			}
		})
	}
}
