package validator

// Mode is a sealed interface representing the validation mode.
// Implemented by ModeAll, ModeModified, and ModeSingle.
type Mode interface {
	validationMode()
}

type ModeAll struct{}
type ModeModified struct{}
type ModeSingle struct {
	ResourceType string
	TargetID     string
}

func (ModeAll) validationMode()      {}
func (ModeModified) validationMode() {}
func (ModeSingle) validationMode()   {}
