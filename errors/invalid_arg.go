package errors

type InvalidArgError struct {
	Subj   string
	Reason string
}

func (e InvalidArgError) Error() string {
	s := ""

	if e.Subj != "" {
		s += e.Subj

		if e.Reason != "" {
			s += ": " + e.Reason
		} else {
			s = "invalid " + s
		}
	}

	return s
}

func NewInvalidArg(subj, reason string) error {
	return InvalidArgError{
		Subj:   subj,
		Reason: reason,
	}
}

func NewRequired(subj string) error {
	return InvalidArgError{
		Subj:   subj,
		Reason: "required",
	}
}
