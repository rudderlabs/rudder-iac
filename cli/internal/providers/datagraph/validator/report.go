package validator

import (
	"fmt"
	"slices"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
)

// RunStatus indicates whether validation was executed or skipped
type RunStatus int

const (
	RunStatusExecuted    RunStatus = iota
	RunStatusNoResources
)

// ResourceValidation holds the validation result for a single resource
type ResourceValidation struct {
	ID           string
	URN          string
	DisplayName  string
	ResourceType string // "model" or "relationship"
	Issues       []dgClient.ValidationIssue
	Err          error
}

// HasErrors returns true if there are any error-severity issues or an execution error
func (rv *ResourceValidation) HasErrors() bool {
	if rv.Err != nil {
		return true
	}
	for _, issue := range rv.Issues {
		if issue.Severity == "error" {
			return true
		}
	}
	return false
}

// HasWarnings returns true if there are any warning-severity issues
func (rv *ResourceValidation) HasWarnings() bool {
	for _, issue := range rv.Issues {
		if issue.Severity == "warning" {
			return true
		}
	}
	return false
}

// completionError returns a non-nil error if the validation has any issues or errors,
// used to signal task failure in progress reporting.
func (rv *ResourceValidation) completionError() error {
	if rv.Err != nil {
		return rv.Err
	}
	if len(rv.Issues) > 0 {
		return fmt.Errorf("validation issues found")
	}
	return nil
}

// ValidationReport holds all validation results for a run
type ValidationReport struct {
	Status    RunStatus
	Resources []*ResourceValidation
}

// HasFailures returns true if any resource has errors
func (vr *ValidationReport) HasFailures() bool {
	for _, r := range vr.Resources {
		if r.HasErrors() {
			return true
		}
	}
	return false
}

// ErrorCount returns the number of resources with errors
func (vr *ValidationReport) ErrorCount() int {
	count := 0
	for _, r := range vr.Resources {
		if r.HasErrors() {
			count++
		}
	}
	return count
}

// WarningCount returns the number of resources with warnings (but no errors)
func (vr *ValidationReport) WarningCount() int {
	count := 0
	for _, r := range vr.Resources {
		if !r.HasErrors() && r.HasWarnings() {
			count++
		}
	}
	return count
}

// PassCount returns the number of resources that passed validation
func (vr *ValidationReport) PassCount() int {
	count := 0
	for _, r := range vr.Resources {
		if !r.HasErrors() && !r.HasWarnings() {
			count++
		}
	}
	return count
}

// ResourcesByType returns resources filtered by type, sorted by URN
func (vr *ValidationReport) ResourcesByType(resourceType string) []*ResourceValidation {
	var result []*ResourceValidation
	for _, r := range vr.Resources {
		if r.ResourceType == resourceType {
			result = append(result, r)
		}
	}
	slices.SortFunc(result, func(a, b *ResourceValidation) int {
		if a.URN < b.URN {
			return -1
		}
		if a.URN > b.URN {
			return 1
		}
		return 0
	})
	return result
}
