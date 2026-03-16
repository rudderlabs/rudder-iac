package cmderrors

// SilentError wraps an error that should cause a non-zero exit code without
// printing an error message to stderr. This is useful for commands that produce
// structured output (e.g., JSON) where the output already contains all failure
// information and an additional stderr message would be redundant or disruptive
// to machine-readable output.
type SilentError struct {
	Err error
}

func (e *SilentError) Error() string {
	return e.Err.Error()
}

func (e *SilentError) Unwrap() error {
	return e.Err
}
