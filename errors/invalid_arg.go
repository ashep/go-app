package errors

type InvalidArgError struct {
	Arg    string
	Reason string
}

func (e InvalidArgError) Error() string {
	if e.Arg != "" && e.Reason == "" {
		return e.Arg + ": invalid"
	}

	if e.Arg == "" && e.Reason != "" {
		return e.Reason
	}

	if e.Arg != "" && e.Reason != "" {
		return e.Arg + ": " + e.Reason
	}

	return "<empty>"
}

func NewInvalidArg(arg, reason string) error {
	return InvalidArgError{
		Arg:    arg,
		Reason: reason,
	}
}

func NewRequiredArg(subj string) error {
	return InvalidArgError{
		Arg:    subj,
		Reason: "required",
	}
}
