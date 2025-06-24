package scene

type Error struct {
	Scene
	Error        string
	SupportEmail string
	Support      string
}

// Create an error scene with the given error message.
func (s Scene) Error(err error) *Error {
	e := &Error{
		Scene:        s,
		SupportEmail: "",
		Support:      "",
	}

	if err != nil {
		e.Error = err.Error()
	}

	return e
}
