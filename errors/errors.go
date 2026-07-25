package errors

type NotFoundError struct {
	Subj string
}

func (e NotFoundError) Error() string {
	return e.Subj + " is not found"
}

type AlreadyExistsError struct {
	Subj string
}

func (e AlreadyExistsError) Error() string {
	return e.Subj + " is already exists"
}

type AccessDeniedError struct{}

func (e AccessDeniedError) Error() string {
	return "access denied"
}

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
