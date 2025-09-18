package typer

import (
	"testing"
)

func TestNewCmdTyper(t *testing.T) {
	t.Run("Command Properties", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			property string
			want     string
			notEmpty bool
		}{
			{"command use", "Use", "typer", false},
			{"command short description", "Short", "", true},
			{"command long description", "Long", "", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cmd := NewCmdTyper()
				if cmd == nil {
					t.Fatal("NewCmdTyper() returned nil")
				}

				var got string
				switch tt.property {
				case "Use":
					got = cmd.Use
				case "Short":
					got = cmd.Short
				case "Long":
					got = cmd.Long
				}

				if tt.notEmpty {
					if got == "" {
						t.Errorf("Expected %s to not be empty", tt.property)
					}
				} else {
					if got != tt.want {
						t.Errorf("Expected %s to be '%s', got '%s'", tt.property, tt.want, got)
					}
				}
			})
		}
	})

	t.Run("Flag Existence", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name        string
			flagName    string
			shortFlag   string
			shouldExist bool
			isRequired  bool
		}{
			{"tracking-plan-id flag", "tracking-plan-id", "t", true, true},
			{"platform flag", "platform", "p", true, true},
			{"output flag", "output", "o", true, true},
			{"non-existent flag", "invalid-flag", "", false, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cmd := NewCmdTyper()
				flag := cmd.Flags().Lookup(tt.flagName)
				exists := flag != nil

				if exists != tt.shouldExist {
					t.Errorf("Flag %s existence = %v, want %v", tt.flagName, exists, tt.shouldExist)
					return
				}

				if tt.shouldExist && tt.shortFlag != "" {
					if flag.Shorthand != tt.shortFlag {
						t.Errorf("Flag %s shorthand = %s, want %s", tt.flagName, flag.Shorthand, tt.shortFlag)
					}
				}
			})
		}
	})

	t.Run("Flag Validation", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name           string
			trackingPlanID string
			platform       string
			outputPath     string
			expectError    bool
			wantError      string
		}{
			{
				name:           "all flags provided",
				trackingPlanID: "tp_123",
				platform:       "kotlin",
				outputPath:     "./output",
				expectError:    false,
			},
			{
				name:           "missing tracking-plan-id",
				trackingPlanID: "",
				platform:       "kotlin",
				outputPath:     "./output",
				expectError:    true,
				wantError:      "tracking plan ID is required (use --tracking-plan-id flag)",
			},
			{
				name:           "missing platform",
				trackingPlanID: "tp_123",
				platform:       "",
				outputPath:     "./output",
				expectError:    true,
				wantError:      "platform is required (use --platform flag)",
			},
			{
				name:           "missing output path",
				trackingPlanID: "tp_123",
				platform:       "kotlin",
				outputPath:     "",
				expectError:    true,
				wantError:      "output path is required (use --output flag)",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cmd := NewCmdTyper()

				// Set flags
				cmd.Flags().Set("tracking-plan-id", tt.trackingPlanID)
				cmd.Flags().Set("platform", tt.platform)
				cmd.Flags().Set("output", tt.outputPath)

				err := cmd.PreRunE(cmd, []string{})

				if tt.expectError {
					if err == nil {
						t.Error("Expected error but got none")
						return
					}
					if tt.wantError != "" && err.Error() != tt.wantError {
						t.Errorf("Expected error '%s', got '%s'", tt.wantError, err.Error())
					}
				} else {
					if err != nil {
						t.Errorf("Expected no error but got: %v", err)
					}
				}
			})
		}
	})
}
