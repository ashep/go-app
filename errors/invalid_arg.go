package errors

type InvalidArgError struct {
	Arg    string
	Reason string
}

func (e InvalidArgError) Error() string {
	s := ""

	if e.Arg != "" {
		s += e.Arg

		if e.Reason != "" {
			s += ": " + e.Reason
		} else {
			s = "invalid " + s
		}
	}

	return s
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
