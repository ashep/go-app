package cfgloader

import (
	"sort"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

type FileNotFoundError struct {
	E string
}

func (e FileNotFoundError) Error() string {
	return e.E
}

func (e FileNotFoundError) Is(target error) bool {
	_, ok := target.(FileNotFoundError)
	return ok
}

type SchemaValidationError struct {
	Result *gojsonschema.Result
}

func (e SchemaValidationError) Error() string {
	if e.Result == nil {
		return "result is nil"
	}

	r := make([]string, 0)
	for _, v := range e.Result.Errors() {
		r = append(r, v.String())
	}

	sort.Strings(r)

	return strings.Join(r, "; ")
}

func (e SchemaValidationError) Is(target error) bool {
	_, ok := target.(SchemaValidationError)
	return ok
}
