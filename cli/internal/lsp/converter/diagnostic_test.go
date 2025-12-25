package converter

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/engine"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/location"
)

func TestEngineDiagnosticsToProtocol(t *testing.T) {
	tests := []struct {
		name        string
		engineDiags []engine.Diagnostic
		wantCount   int
	}{
		{
			name:        "empty diagnostics",
			engineDiags: []engine.Diagnostic{},
			wantCount:   0,
		},
		{
			name: "single diagnostic",
			engineDiags: []engine.Diagnostic{
				{
					File:     "test.yaml",
					Rule:     "test/rule",
					Severity: validation.SeverityError,
					Message:  "test error",
					Position: location.Position{Line: 1, Column: 1},
				},
			},
			wantCount: 1,
		},
		{
			name: "multiple diagnostics",
			engineDiags: []engine.Diagnostic{
				{
					File:     "test1.yaml",
					Severity: validation.SeverityError,
					Message:  "error 1",
					Position: location.Position{Line: 1, Column: 1},
				},
				{
					File:     "test2.yaml",
					Severity: validation.SeverityWarning,
					Message:  "warning 1",
					Position: location.Position{Line: 5, Column: 10},
				},
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EngineDiagnosticsToProtocol(tt.engineDiags)
			if len(got) != tt.wantCount {
				t.Errorf("EngineDiagnosticsToProtocol() returned %d diagnostics, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestEngineDiagnosticToProtocol(t *testing.T) {
	tests := []struct {
		name       string
		engineDiag engine.Diagnostic
		wantLine   protocol.UInteger
		wantChar   protocol.UInteger
		wantSev    protocol.DiagnosticSeverity
	}{
		{
			name: "error severity with position",
			engineDiag: engine.Diagnostic{
				File:     "test.yaml",
				Rule:     "test/rule",
				Severity: validation.SeverityError,
				Message:  "test error",
				Position: location.Position{Line: 10, Column: 5},
			},
			wantLine: 9, // 1-based to 0-based
			wantChar: 4, // 1-based to 0-based
			wantSev:  protocol.DiagnosticSeverityError,
		},
		{
			name: "warning severity",
			engineDiag: engine.Diagnostic{
				Severity: validation.SeverityWarning,
				Message:  "test warning",
				Position: location.Position{Line: 1, Column: 1},
			},
			wantLine: 0,
			wantChar: 0,
			wantSev:  protocol.DiagnosticSeverityWarning,
		},
		{
			name: "info severity",
			engineDiag: engine.Diagnostic{
				Severity: validation.SeverityInfo,
				Message:  "test info",
				Position: location.Position{Line: 1, Column: 1},
			},
			wantLine: 0,
			wantChar: 0,
			wantSev:  protocol.DiagnosticSeverityInformation,
		},
		{
			name: "zero position (defaults to 0, 0)",
			engineDiag: engine.Diagnostic{
				Severity: validation.SeverityError,
				Message:  "test",
				Position: location.Position{Line: 0, Column: 0},
			},
			wantLine: 0,
			wantChar: 0,
			wantSev:  protocol.DiagnosticSeverityError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engineDiagnosticToProtocol(tt.engineDiag)

			// Check severity
			if got.Severity == nil {
				t.Error("Severity should not be nil")
			} else if *got.Severity != tt.wantSev {
				t.Errorf("Severity = %v, want %v", *got.Severity, tt.wantSev)
			}

			// Check position
			if got.Range.Start.Line != tt.wantLine {
				t.Errorf("Start.Line = %v, want %v", got.Range.Start.Line, tt.wantLine)
			}
			if got.Range.Start.Character != tt.wantChar {
				t.Errorf("Start.Character = %v, want %v", got.Range.Start.Character, tt.wantChar)
			}

			// Check message
			if got.Message != tt.engineDiag.Message {
				t.Errorf("Message = %v, want %v", got.Message, tt.engineDiag.Message)
			}

			// Check source
			if got.Source == nil {
				t.Error("Source should not be nil")
			} else if *got.Source != "rudder-validator" {
				t.Errorf("Source = %v, want 'rudder-validator'", *got.Source)
			}

			// Check code (if rule is present)
			if tt.engineDiag.Rule != "" {
				if got.Code == nil {
					t.Error("Code should not be nil when Rule is present")
				}
			}
		})
	}
}

func TestEngineDiagnosticWithFragment(t *testing.T) {
	engineDiag := engine.Diagnostic{
		Severity: validation.SeverityError,
		Message:  "test error",
		Position: location.Position{Line: 5, Column: 10},
		Fragment: "testField",
	}

	got := engineDiagnosticToProtocol(engineDiag)

	// With fragment, end character should be start + fragment length
	wantStartLine := protocol.UInteger(4)   // 5 - 1
	wantStartChar := protocol.UInteger(9)    // 10 - 1
	wantEndChar := wantStartChar + protocol.UInteger(len("testField"))

	if got.Range.Start.Line != wantStartLine {
		t.Errorf("Start.Line = %v, want %v", got.Range.Start.Line, wantStartLine)
	}
	if got.Range.Start.Character != wantStartChar {
		t.Errorf("Start.Character = %v, want %v", got.Range.Start.Character, wantStartChar)
	}
	if got.Range.End.Character != wantEndChar {
		t.Errorf("End.Character = %v, want %v", got.Range.End.Character, wantEndChar)
	}
}

func TestSeverityToProtocol(t *testing.T) {
	tests := []struct {
		name     string
		severity validation.Severity
		want     protocol.DiagnosticSeverity
	}{
		{
			name:     "error severity",
			severity: validation.SeverityError,
			want:     protocol.DiagnosticSeverityError,
		},
		{
			name:     "warning severity",
			severity: validation.SeverityWarning,
			want:     protocol.DiagnosticSeverityWarning,
		},
		{
			name:     "info severity",
			severity: validation.SeverityInfo,
			want:     protocol.DiagnosticSeverityInformation,
		},
		{
			name:     "unknown severity defaults to error",
			severity: validation.Severity("unknown"),
			want:     protocol.DiagnosticSeverityError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := severityToProtocol(tt.severity)
			if got != tt.want {
				t.Errorf("severityToProtocol() = %v, want %v", got, tt.want)
			}
		})
	}
}
