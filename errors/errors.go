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
