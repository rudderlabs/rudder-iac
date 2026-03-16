package validator

// Displayer formats and renders a validation report.
type Displayer interface {
	Display(report *ValidationReport)
}
