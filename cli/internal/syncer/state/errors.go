package state

type ErrIncompatibleVersion struct {
	Version string
}

func (e *ErrIncompatibleVersion) Error() string {
	return "incompatible state version: " + e.Version
}

type ErrURNAlreadyExists struct {
	URN string
}

func (e *ErrURNAlreadyExists) Error() string {
	return "URN already exists: " + e.URN
}
