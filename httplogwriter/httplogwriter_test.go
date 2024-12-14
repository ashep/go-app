package httplogwriter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ashep/go-app/httplogwriter"
)

func TestWriter_New(tt *testing.T) {
	tt.Run("EmptyURL", func(t *testing.T) {
		_, err := httplogwriter.New("", "", "")
		assert.EqualError(tt, err, "empty url")
	})

	tt.Run("InvalidURL", func(t *testing.T) {
		_, err := httplogwriter.New(string([]byte{0x0}), "", "")
		assert.EqualError(tt, err, `invalid url: parse "\x00": net/url: invalid control character in URL`)
	})

	tt.Run("Ok", func(t *testing.T) {
		_, err := httplogwriter.New("anURL", "", "")
		assert.NoError(tt, err)
	})
}
