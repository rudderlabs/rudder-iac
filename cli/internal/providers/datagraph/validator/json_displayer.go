package validator

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSONDisplayer renders a validation report as JSON.
type JSONDisplayer struct {
	w io.Writer
}

// NewJSONDisplayer creates a new JSONDisplayer that writes to w.
func NewJSONDisplayer(w io.Writer) *JSONDisplayer {
	return &JSONDisplayer{w: w}
}

// Display renders the validation report as JSON.
func (d *JSONDisplayer) Display(report *ValidationReport) {
	type jsonIssue struct {
		Rule     string `json:"rule"`
		Severity string `json:"severity"`
		Message  string `json:"message"`
	}
	type jsonResource struct {
		ID           string      `json:"id"`
		URN          string      `json:"urn"`
		DisplayName  string      `json:"displayName"`
		ResourceType string      `json:"resourceType"`
		Status       string      `json:"status"`
		Issues       []jsonIssue `json:"issues,omitempty"`
		Error        string      `json:"error,omitempty"`
	}
	type jsonOutput struct {
		Resources []jsonResource `json:"resources"`
	}

	out := jsonOutput{
		Resources: make([]jsonResource, 0, len(report.Resources)),
	}

	for _, rv := range report.Resources {
		jr := jsonResource{
			ID:           rv.ID,
			URN:          rv.URN,
			DisplayName:  rv.DisplayName,
			ResourceType: rv.ResourceType,
		}

		if rv.Err != nil {
			jr.Status = "error"
			jr.Error = rv.Err.Error()
		} else if rv.HasErrors() {
			jr.Status = "failed"
		} else if rv.HasWarnings() {
			jr.Status = "warning"
		} else {
			jr.Status = "passed"
		}

		for _, issue := range rv.Issues {
			jr.Issues = append(jr.Issues, jsonIssue{
				Rule:     issue.Rule,
				Severity: issue.Severity,
				Message:  issue.Message,
			})
		}

		out.Resources = append(out.Resources, jr)
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Fprintln(d.w, string(data))
}
