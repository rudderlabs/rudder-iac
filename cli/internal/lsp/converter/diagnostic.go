package converter

import (
	protocol "github.com/tliron/glsp/protocol_3_16"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/engine"
)

// EngineDiagnosticsToProtocol converts validation engine diagnostics to LSP protocol diagnostics
func EngineDiagnosticsToProtocol(engineDiags []engine.Diagnostic) []protocol.Diagnostic {
	protocolDiags := make([]protocol.Diagnostic, 0, len(engineDiags))

	for _, engineDiag := range engineDiags {
		protocolDiag := engineDiagnosticToProtocol(engineDiag)
		protocolDiags = append(protocolDiags, protocolDiag)
	}

	return protocolDiags
}

// engineDiagnosticToProtocol converts a single engine diagnostic to protocol diagnostic
func engineDiagnosticToProtocol(engineDiag engine.Diagnostic) protocol.Diagnostic {
	// Convert severity from string to protocol severity int
	severity := severityToProtocol(engineDiag.Severity)

	// Convert position from 1-based to 0-based
	var (
		line      protocol.UInteger = 0
		character protocol.UInteger = 0
	)

	if engineDiag.Position.Line > 0 {
		line = protocol.UInteger(engineDiag.Position.Line - 1)
	}

	if engineDiag.Position.Column > 0 {
		character = protocol.UInteger(engineDiag.Position.Column - 1)
	}

	// Create range for the diagnostic
	// If we have a fragment, we can estimate the end position
	var (
		endLine      protocol.UInteger = line
		endCharacter protocol.UInteger = character
	)

	// TODO: Fix the fragment length calculation as by default it will be character + 1
	if engineDiag.Fragment != "" {
		// Estimate end character based on fragment length
		endCharacter = character + protocol.UInteger(len(engineDiag.Fragment))
	} else {
		// Default to single character range
		endCharacter = character + 1
	}

	diagRange := protocol.Range{
		Start: protocol.Position{
			Line:      line,
			Character: character,
		},
		End: protocol.Position{
			Line:      endLine,
			Character: endCharacter,
		},
	}

	source := "rudder-validator"
	protocolDiag := protocol.Diagnostic{
		Range:    diagRange,
		Severity: &severity,
		Message:  engineDiag.Message,
		Source:   &source,
	}

	// Add rule ID as code if present
	if engineDiag.Rule != "" {
		code := protocol.IntegerOrString{Value: engineDiag.Rule}
		protocolDiag.Code = &code
	}

	return protocolDiag
}

// severityToProtocol converts validation severity string to protocol severity int
func severityToProtocol(severity validation.Severity) protocol.DiagnosticSeverity {
	switch severity {
	case validation.SeverityError:
		return protocol.DiagnosticSeverityError // 1
	case validation.SeverityWarning:
		return protocol.DiagnosticSeverityWarning // 2
	case validation.SeverityInfo:
		return protocol.DiagnosticSeverityInformation // 3
	default:
		return protocol.DiagnosticSeverityError
	}
}
