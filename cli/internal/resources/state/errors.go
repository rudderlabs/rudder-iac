package state

type ErrURNAlreadyExists struct {
	URN string
}

func (e *ErrURNAlreadyExists) Error() string {
	return "URN already exists: " + e.URN
}
